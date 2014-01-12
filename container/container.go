package container

import (
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"time"
	"launchpad.net/goyaml"
)

type Container struct {
	Name         string
	Type         string
	InstanceType string
	Running      bool
	Id           string
	Ip           string
	Links        []string
	Dependencies []*Container
	Ports        map[string]string
	Volumes      map[string]string
	Custom       map[string]interface{}
}

type inspection struct {
	NetworkSettings map[string]string "NetworkSettings,omitempty"
}

func (c *Container) init() {
	if c.Ports == nil {
		c.Ports = make(map[string]string)
	}
	if c.Links == nil {
		c.Links = make([]string, 0)
	}
	if c.Dependencies == nil {
		c.Dependencies = make([]*Container, 0)
	}
}

func (c *Container) Clean(done chan bool) {
	fmt.Println("Cleaning", c.Name, "...")

	cmd, _ := exec.LookPath("docker")

	killCmd := exec.Command(cmd, "kill", c.Name)
	killCmd.Output()

	removeCmd := exec.Command(cmd, "rm", c.Name)
	removeCmd.Output()

	done <- true
}

func (c *Container) Build(done chan bool) {
	cmd, _ := exec.LookPath("docker")
	buildCmd := exec.Command(cmd, "build", "-rm", "-t", "arch_o_matic/"+c.Name, "/tmp/arch-o-matic/"+c.Name)

	fmt.Println("Building", c.Name, "...")

	out, err := buildCmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}

	done <- true
}

func (c *Container) IsRunning() bool {
	return c.Running
}

func (c *Container) IsReady() bool {
	ready := true

	for _, dependency := range c.Dependencies {
		ready = ready && dependency.IsRunning()
	}

	return ready
}

func (c *Container) AddDependency(container *Container) {
	c.Dependencies = append(c.Dependencies, container)
}

func (c *Container) Start() {
	c.init()

	cmd, _ := exec.LookPath("docker")
	runFunc := reflect.ValueOf(exec.Command)
	rawArgs := []string{cmd, "run", "-d", "-name", c.Name}

	// Add links
	for _, link := range c.Links {
		rawArgs = append(rawArgs, "--link="+link+":"+link)
	}

	// Add ports
	for portIn, portOut := range c.Ports {
		rawArgs = append(rawArgs, "-p="+string(portIn)+":"+string(portOut))
	}

	// Add volumes
	for volumeHost, volumeContainer := range c.Volumes {
		rawArgs = append(rawArgs, "-v="+volumeHost+":"+volumeContainer)
	}

	rawArgs = append(rawArgs, "arch_o_matic/"+c.Name)

	// Initiate the command with several arguments
	runCmd := runFunc.Call(buildArguments(rawArgs))[0].Interface().(*exec.Cmd)

	out, err := runCmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}

	c.Id = strings.TrimSpace(string(out))
	c.Running = true

	time.Sleep(2 * time.Second)
	c.retrieveIp()

	fmt.Println("Container", c.Name, "started", "(" +c.Ip+":"+c.GetFirstPort()+") :", c.Id)
}

func (c *Container) GetCustomValue(name string) interface{} {
	return c.Custom[name]
}

func (c *Container) GetCustomValueAsString(name string) string {
	return c.Custom[name].(string)
}

func (c *Container) GetFirstPort() string {
	keys := make([]string, 0)
	for _, key := range c.Ports {
		keys = append(keys, key)
	}

	return c.Ports[keys[0]]
}

func (c *Container) Stop() {
	cmd, _ := exec.LookPath("docker")

	killCmd := exec.Command(cmd, "kill", c.Id)
	killCmd.Run()

	c.Running = false
}

func (c *Container) retrieveIp () {
	cmd, _ := exec.LookPath("docker")

	inspectCmd := exec.Command(cmd, "inspect", c.Id)
	out, err := inspectCmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}

	var results []inspection
	goyaml.Unmarshal(out, &results)

	c.Ip = results[0].NetworkSettings["IPAddress"]
}

func buildArguments(rawArgs []string) []reflect.Value {
	args := make([]reflect.Value, 0)

	for _, arg := range rawArgs {
		args = append(args, reflect.ValueOf(arg))
	}

	return args
}

