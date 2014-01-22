package maestro

import (
	"bytes"
	"github.com/marmelab/gaudi/container"
	"github.com/marmelab/gaudi/util"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"path/filepath"
	"text/template"
	"strings"
)

type Maestro struct {
	Containers map[string]*container.Container
}

type TemplateData struct {
	Maestro   *Maestro
	Container *container.Container
}

func (m *Maestro) InitFromFile(file string) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	m.InitFromString(string(content), filepath.Dir(file))
}

func (maestro *Maestro) InitFromString(content, relativePath string) {
	err := goyaml.Unmarshal([]byte(content), &maestro)
	if err != nil {
		panic(err)
	}
	if maestro.Containers == nil {
		panic("No container to start")
	}

	// Fill name & dependencies
	for name := range maestro.Containers {
		currentContainer := maestro.Containers[name]
		currentContainer.Name = name

		for _, dependency := range currentContainer.Links {
			currentContainer.AddDependency(maestro.Containers[dependency])
		}

		// Add relative path to volumes
		for volumeHost, volumeContainer := range currentContainer.Volumes {
			if string(volumeHost[0]) != "/" {
				delete(currentContainer.Volumes, volumeHost)

				if !util.IsDir(relativePath+"/"+volumeHost) {
					panic(relativePath+"/"+volumeHost+" should be a directory")
				}

				currentContainer.Volumes[relativePath+"/"+volumeHost] = volumeContainer
			} else if !util.IsDir(volumeHost) {
				panic(volumeHost+" should be a directory")
			}
		}
	}
}

func (maestro *Maestro) parseTemplates() {
	// Running withmock doesn't include templates files in withmock's temporary dir
	path := os.Getenv("GOPATH")
	testPath := os.Getenv("ORIG_GOPATH")
	if len(testPath) > 0 {
		path = testPath
	}

	templateDir := path + "/src/github.com/marmelab/gaudi/templates/"
	parsedTemplateDir := "/tmp/gaudi/"
	templateData := TemplateData{maestro, nil}
	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToLower": strings.ToLower,
	}

	err := os.MkdirAll(parsedTemplateDir, 0700)
	if err != nil {
		panic(err)
	}

	for _, currentContainer := range maestro.Containers {
		files, err := ioutil.ReadDir(templateDir + currentContainer.Type)
		if err != nil {
			continue
		}

		err = os.MkdirAll(parsedTemplateDir+currentContainer.Name, 0755)
		if err != nil {
			panic(err)
		}

		// Parse & copy files
		for _, file := range files {
			destination := parsedTemplateDir + currentContainer.Name + "/" + file.Name()
			if file.IsDir() {
				err := os.MkdirAll(destination, 0755)
				if err != nil {
					panic(err)
				}

				continue
			}

			// Read the template
			filePath := templateDir + currentContainer.Type + "/" + file.Name()
			content, err := ioutil.ReadFile(filePath)
			if err != nil {
				panic(err)
			}

			// Parse it (we need to change default delimiters because sometimes we have to parse values like ${{{ .Val }}}
			// which cause an error)
			tmpl, err := template.New(filePath).Funcs(funcMap).Delims("[[", "]]").Parse(string(content))
			if err != nil {
				panic(err)
			}

			templateData.Container = currentContainer
			var result bytes.Buffer
			err = tmpl.Execute(&result, templateData)
			if err != nil {
				panic(err)
			}

			// Create new file
			ioutil.WriteFile(destination, []byte(result.String()), 0644)
		}
	}
}

func (maestro *Maestro) Start(rebuild bool) {
	rebuild = rebuild || !maestro.HasParsedTemplates()

	if rebuild {
		maestro.parseTemplates()

		cleanChans := make(chan bool, len(maestro.Containers))
		// Clean all containers
		for _, currentContainer := range maestro.Containers {
			go currentContainer.Clean(cleanChans)
		}
		<-cleanChans


		buildChans := make(chan bool, len(maestro.Containers))

		// Build all containers
		for _, currentContainer := range maestro.Containers {
			go currentContainer.Build(buildChans)
		}
		<-buildChans
	}

	startChans := make(map[string]chan bool)

	// Start all containers
	for name, currentContainer := range maestro.Containers {
		startChans[name] = make(chan bool)

		go maestro.startContainer(currentContainer, startChans)
	}

	// Waiting for all containers to start
	for containerName, _ := range maestro.Containers {
		<-startChans[containerName]
	}
}

func (maestro *Maestro) GetContainer(name string) *container.Container {
	return maestro.Containers[name]
}

func (maestro *Maestro) Check () {
	for _, currentContainer := range maestro.Containers {
		currentContainer.CheckIfRunning()
	}
}

func (maestro *Maestro) Stop() {
	killChans := make(chan bool, len(maestro.Containers))

	for _, currentContainer := range maestro.Containers {
		go currentContainer.Kill(killChans, false)
	}

	<-killChans
}

func (maestro *Maestro) startContainer(currentContainer *container.Container, done map[string]chan bool) {
	// Waiting for dependencies to start
	for _, dependency := range currentContainer.Dependencies {
		<-done[dependency.Name]
	}

	currentContainer.Start()

	close(done[currentContainer.Name])
}

func (maestro *Maestro) HasParsedTemplates () bool {
	parsedTemplateDir := "/tmp/gaudi/"

	for containerName := range maestro.Containers {
		if !util.IsDir(parsedTemplateDir+containerName) {
			return false
		}
	}

	return true
}
