package containerCollection

import (
	"github.com/marmelab/gaudi/container"
	"github.com/marmelab/gaudi/util"
	"os"
)

type ContainerCollection map[string]*container.Container

func Merge(c1, c2 ContainerCollection) ContainerCollection {
	result := make(ContainerCollection)

	for name, currentContainer := range c1 {
		result[name] = currentContainer
	}
	for name, currentContainer := range c2 {
		result[name] = currentContainer
	}

	return result
}

func (collection ContainerCollection) Init(relativePath string) bool {
	hasGaudiManagedContainer := false

	// Fill name & dependencies
	for name, currentContainer := range collection {
		currentContainer.Name = name
		currentContainer.Init()

		if currentContainer.IsGaudiManaged() {
			hasGaudiManagedContainer = true
			currentContainer.Image = "gaudi/" + name
		}

		for _, dependency := range currentContainer.Links {
			if depContainer, exists := collection[dependency]; exists {
				currentContainer.AddDependency(depContainer)
			} else {
				util.LogError(name + " references a non existing application : " + dependency)
			}
		}

		// Add relative path to volumes
		for volumeHost, volumeContainer := range currentContainer.Volumes {
			// Relative volume host
			if string(volumeHost[0]) != "/" {
				delete(currentContainer.Volumes, volumeHost)
				volumeHost = relativePath + "/" + volumeHost

				currentContainer.Volumes[volumeHost] = volumeContainer
			}

			// Create directory if needed
			if !util.IsDir(volumeHost) {
				err := os.MkdirAll(volumeHost, 0755)
				if err != nil {
					util.LogError(err)
				}
			}
		}
	}

	return hasGaudiManagedContainer
}

func (collection ContainerCollection) Get(args ...interface{}) *container.Container {
	if c, ok := collection[args[0].(string)]; ok {
		return c
	} else if len(args) == 2 {
		return args[1].(*container.Container)
	}

	return nil
}

func (collection ContainerCollection) GetType(containerType string) *container.Container {
	for _, currentContainer := range collection {
		if currentContainer.Type == containerType {
			return currentContainer
		}
	}

	return nil
}

func (collection ContainerCollection) Start(rebuild bool) {
	collection.CheckIfNotEmpty()

	if rebuild {
		collection.Clean()
		collection.Build()
	}

	startChans := make(map[string]chan bool, len(collection))

	// Start all applications
	for name, currentContainer := range collection {
		startChans[name] = make(chan bool)

		go startOne(currentContainer, rebuild, startChans)
	}

	// Waiting for all applications to start
	for name, _ := range collection {
		<-startChans[name]
	}
}

func (collection ContainerCollection) Stop() {
	collection.CheckIfNotEmpty()

	nbContainers := len(collection)
	killChans := make(chan bool, nbContainers)

	for _, currentContainer := range collection {
		go currentContainer.Kill(false, killChans)
	}

	// Waiting for all applications to stop
	waitForIt(killChans)
}

func (collection ContainerCollection) CheckIfNotEmpty() {
	// Check if there is at least a container
	if collection == nil || len(collection) == 0 {
		util.LogError("Gaudi requires at least an application to be defined to start anything")
	}
}

func (collection ContainerCollection) Clean() {
	nbContainers := len(collection)
	cleanChans := make(chan bool, nbContainers)

	// Clean all applications
	for _, currentContainer := range collection {
		go currentContainer.Clean(cleanChans)
	}
	waitForIt(cleanChans)
}

func (collection ContainerCollection) Build() {
	nbContainers := len(collection)
	buildChans := make(chan bool, nbContainers)

	// Build all
	for _, currentContainer := range collection {
		go currentContainer.BuildOrPull(buildChans)
	}
	waitForIt(buildChans)
}

func (collection ContainerCollection) IsComponentDependingOf(container *container.Container, otherComponentType string) bool {

	for _, currentContainer := range collection {
		if currentContainer.Type == otherComponentType && currentContainer.DependsOf(container.Type) {
			return true
		}
	}

	return false
}

func waitForIt(channels chan bool) {
	nbContainers := cap(channels)

	for i := 0; i < nbContainers; i++ {
		<-channels
	}
}

func (collection ContainerCollection) AddAmbassadors() {
	for name, currentContainer := range collection {
		if currentContainer.Ambassador.Type == "" {
			continue
		}

		// Add the ambassador
		ambassadorName := "ambassador-" + name
		ambassador := &container.Container{Name: ambassadorName, Type: "ambassador"}
		ambassador.Init()

		ambassador.Links = append(ambassador.Links, name)
		ambassador.Ports[currentContainer.Ambassador.Port] = currentContainer.Ambassador.Port

		collection[ambassadorName] = ambassador
	}
}

func startOne(currentContainer *container.Container, rebuild bool, done map[string]chan bool) {
	// Waiting for dependencies to be started
	for _, dependency := range currentContainer.Dependencies {
		if dependency.Name == currentContainer.Name {
			util.LogError("Application " + currentContainer.Name + " can't be linked with itself.")
		}

		<-done[dependency.Name]
	}

	currentContainer.Start(rebuild)

	close(done[currentContainer.Name])
}
