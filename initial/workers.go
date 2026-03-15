package initial

import (
	"igloo/data"
	"sync"
)

func dirWorker(readJobs <-chan string, wg *sync.WaitGroup, theWorks *data.CollectedInfo) {
	defer wg.Done()

	for path := range readJobs {
		readDir(path, theWorks, false)
	}
}

func fileWorker(readJobs <-chan string, wg *sync.WaitGroup, theWorks *data.CollectedInfo, config *data.Config) {
	defer wg.Done()
	for path := range readJobs {
		readFile(path, theWorks, config)
	}
}
