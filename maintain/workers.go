package maintain

import (
	"database/sql"
	"log"
	"igloo/data"
	"sync"
)

func scanWorker(scanJobs <-chan data.EntryHeader, readJobs chan<- data.SyncJob, uniqueMappedEntries map[string]data.EntryHeader, wg *sync.WaitGroup, config *data.Config) {
	defer wg.Done()
	for job := range scanJobs {
		err := scanUpdatedDir(readJobs, job.Path, uniqueMappedEntries, config)
		if err != nil {
			log.Fatal(err)
		}
	}
}
func readWorker(readJobs <-chan data.SyncJob, con *sql.DB, wg *sync.WaitGroup, config *data.Config) {
	defer wg.Done()
	for job := range readJobs {
		readEntry(job, config, con)
	}
}
func newDirWorker(newDirJobs <-chan string, readJobs chan<- data.SyncJob, con *sql.DB, wg *sync.WaitGroup, config *data.Config) {
	defer wg.Done()
	for path := range newDirJobs {
		err := traverseNewDir(readJobs, path, config, con)
		if err != nil {
			log.Fatal(err)
		}
	}
}
func deletionWorker(delJobs <-chan string, con *sql.DB, wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range delJobs {
		err := checkDelete(path, con)
		if err != nil {
			log.Fatal(err)
		}
	}
}
