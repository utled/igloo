package maintain

import (
	"fmt"
	"igloo/data"
	"sync"
)

func scanWorker(scanJobs <-chan data.EntryHeader, readJobs chan<- data.SyncJob, indexedEntries map[string]data.EntryHeader, wg *sync.WaitGroup, config *data.Config) {
	defer wg.Done()
	for job := range scanJobs {
		err := scanUpdatedDir(readJobs, job.Path, indexedEntries, config)
		if err != nil {
			fmt.Println(err)
		}
	}
}
func readWorker(readJobs <-chan data.SyncJob, syncInfo *data.SyncInfo, wg *sync.WaitGroup, config *data.Config) {
	defer wg.Done()
	for job := range readJobs {
		readEntry(job, config, syncInfo)
	}
}
func newDirWorker(newDirJobs <-chan string, readJobs chan<- data.SyncJob, wg *sync.WaitGroup, indexedEntries map[string]data.EntryHeader, config *data.Config) {
	defer wg.Done()
	for path := range newDirJobs {
		err := traverseNewDir(readJobs, path, indexedEntries, config)
		if err != nil {
			fmt.Println(err)
		}
	}
}
func deletionWorker(delJobs <-chan data.DeletionJob, syncInfo *data.SyncInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range delJobs {
		err := checkDelete(path, syncInfo)
		if err != nil {
			fmt.Println(err)
		}
	}
}
