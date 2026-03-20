package initial

import (
	"igloo/data"
	"sync"
)

func dirWorker(readJobs <-chan data.ReadJob, wg *sync.WaitGroup, theWorks *data.CollectedInfo) {
	defer wg.Done()

	for job := range readJobs {
		readDir(job.Path, job.Stat, theWorks, false)
	}
}

func fileWorker(readJobs <-chan data.ReadJob, wg *sync.WaitGroup, theWorks *data.CollectedInfo, config *data.Config) {
	defer wg.Done()
	for job := range readJobs {
		readFile(job.Path, job.Stat, theWorks, config)
	}
}
