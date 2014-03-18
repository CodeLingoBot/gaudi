package docker

import (
	"flag"
	"errors"
	"github.com/marmelab/gaudi/util"
	"os/exec"
	"reflect"
	"strings"
	"time"
)


var (
	docker, _ = exec.LookPath("docker")
	noCache  = flag.Bool("no-cache", false, "Disable build cache")
)

func main() {
	flag.Parse()
}

func ImageExists(name string) bool {
	imagesCmd := exec.Command(docker, "images", name)

	out, err := imagesCmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Retrieve lines & remove first and last one
	lines := strings.Split(string(out), "\n")
	lines = lines[1 : len(lines)-1]

	return len(lines) > 0
}

func Remove(name string) {
	removeCmd := exec.Command(docker, "rm", name)
	removeErr := removeCmd.Start()
	if removeErr != nil {
		util.LogError(removeErr)
	}
	time.Sleep(1 * time.Second)
}

func Kill(name string) {
	killCommand := exec.Command(docker, "kill", name)
	killErr := killCommand.Start()
	if killErr != nil {
		util.LogError(killErr)
	}

	time.Sleep(1 * time.Second)
}

func Build(name, path string) {
	buildFunc := reflect.ValueOf(exec.Command)
	rawArgs := []string{docker, "build", "-t", name, path}

	if *noCache {
		rawArgs = append(rawArgs, "--no-cache")
	}

	util.Debug(rawArgs)

	buildCmd := buildFunc.Call(buildArguments(rawArgs))[0].Interface().(*exec.Cmd)
	out, err := buildCmd.CombinedOutput()
	if err != nil {
		util.LogError(string(out))
	}

	time.Sleep(1 * time.Second)
}

func Pull(name string) {
	pullCmd := exec.Command(docker, "pull", name)
	util.Debug("Pull command:", pullCmd.Args)

	out, err := pullCmd.CombinedOutput()
	if err != nil {
		util.LogError(string(out))
	}
}

/**
 * Start a container as a server
 */
func Start(name, image string, links []string, ports, volumes, environments map[string]string) string {
	runFunc := reflect.ValueOf(exec.Command)
	rawArgs := []string{docker, "run", "-d", "-i", "-t", "--name=" + name}

	// Add environments
	util.Debug(environments)
	for envName, envValue := range environments {
		rawArgs = append(rawArgs, "-e="+envName+"="+envValue)
	}

	// Add links
	for _, link := range links {
		rawArgs = append(rawArgs, "--link="+link+":"+link)
	}

	// Add ports
	for portIn, portOut := range ports {
		rawArgs = append(rawArgs, "-p="+string(portIn)+":"+string(portOut))
	}

	// Add volumes
	for volumeHost, volumeContainer := range volumes {
		rawArgs = append(rawArgs, "-v="+volumeHost+":"+volumeContainer)
	}

	rawArgs = append(rawArgs, image)

	// Initiate the command with several arguments
	runCmd := runFunc.Call(buildArguments(rawArgs))[0].Interface().(*exec.Cmd)
	util.Debug("Start command:", runCmd.Args)

	out, err := runCmd.CombinedOutput()
	if err != nil {
		util.LogError(string(out))
	}

	return string(out)
}

/**
 * Start a container as binary
 */
func Run(name, currentPath string, arguments []string) string {
	runFunc := reflect.ValueOf(exec.Command)
	rawArgs := []string{docker, "run", "-v=" + currentPath + ":" + currentPath, "-w=" + currentPath, name}

	for _, argument := range arguments {
		rawArgs = append(rawArgs, argument)
	}

	runCmd := runFunc.Call(buildArguments(rawArgs))[0].Interface().(*exec.Cmd)
	util.Debug("Run command:", runCmd.Args)

	out, err := runCmd.CombinedOutput()
	if err != nil {
		util.LogError(string(out))
	}

	return string(out)
}

func Inspect(id string) ([]byte, error) {
	inspectCmd := exec.Command(docker, "inspect", id)

	out, err := inspectCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func SnapshotProcesses() (map[string]string, error) {
	images := make(map[string]string)

	psCommand := exec.Command(docker, "ps")
	out, err := psCommand.CombinedOutput()
	if err != nil {
		return nil, errors.New(string(out))
	}

	// Retrieve lines & remove first and last one
	lines := strings.Split(string(out), "\n")
	lines = lines[1 : len(lines)-1]

	for _, line := range lines {
		fields := strings.Fields(line)
		nameParts := strings.Split(fields[1], ":")

		images[nameParts[0]] = fields[0]
	}

	return images, nil
}

func buildArguments(rawArgs []string) []reflect.Value {
	args := make([]reflect.Value, 0)

	for _, arg := range rawArgs {
		args = append(args, reflect.ValueOf(arg))
	}

	return args
}
