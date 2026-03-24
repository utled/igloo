package maintain

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"igloo/config"
	"igloo/data"
	"igloo/db"
)

const (
	deletionJobBufferSize = 50
	scanJobBufferSize     = 100
	readJobBufferSize     = 300
	newDirJobBufferSize   = 50
	deletionWorkers       = 10
	entryScanners         = 15
	entryReaders          = 40
	newDirWorkers         = 5
)

// deletionWorker is responsible for iterating all indexed entries and collect all no longer existing file system entries for deletion
func deletionWorker(delJobs <-chan data.DeletionJob, syncInfo *data.SyncInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range delJobs {
		err := checkDelete(path, syncInfo)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// scanWorker is responsible for scanning updated directories to categorize the update type and produce readjobs of the changed entries
func scanWorker(scanJobs <-chan data.EntryHeader, readJobs chan<- data.SyncJob, indexedEntries map[string]data.EntryHeader, wg *sync.WaitGroup, config *data.Config) {
	defer wg.Done()
	for job := range scanJobs {
		err := scanUpdatedDir(readJobs, job.Path, indexedEntries, config)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// newDirWorker is responsible for traversing newly created directories and categorize new file system entries to produce readjobs of the entries
func newDirWorker(newDirJobs <-chan string, readJobs chan<- data.SyncJob, wg *sync.WaitGroup, indexedEntries map[string]data.EntryHeader, config *data.Config) {
	defer wg.Done()
	for path := range newDirJobs {
		err := traverseNewDir(readJobs, path, indexedEntries, config)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// readWorker is responsible for reading and collecting metadata/contents for new/updated file system entries
func readWorker(readJobs <-chan data.SyncJob, syncInfo *data.SyncInfo, wg *sync.WaitGroup, config *data.Config) {
	defer wg.Done()
	for job := range readJobs {
		readEntry(job, config, syncInfo)
	}
}

// updateAfterSync takes the collected data from the sync processes and triggers db updates of the index
func updateAfterSync(syncInfo *data.SyncInfo, con *sql.DB) {
	countOfDeletions := len(syncInfo.Deletions)
	countOfNewEntries := len(syncInfo.NewEntries)
	countOfUpdatesWContent := len(syncInfo.UpdatesWContent)
	countOfUpdatesWOContent := len(syncInfo.UpdatesWOContent)
	fmt.Printf("Starting DB updates for:\n%d Deletions\n%d New entries\n%d Updates with content\n%d Updates without content\n",
		countOfDeletions,
		countOfNewEntries,
		countOfUpdatesWContent,
		countOfUpdatesWOContent,
	)
	updateDBStart := time.Now()
	if countOfDeletions > 0 {
		data.DeleteEntries(con, syncInfo.Deletions)
	}
	if countOfNewEntries > 0 {
		data.WriteFullEntries(con, syncInfo.NewEntries)
	}
	if countOfUpdatesWContent > 0 {
		data.UpdateEntriesWithContent(con, syncInfo.UpdatesWContent)
	}
	if countOfUpdatesWOContent > 0 {
		data.UpdateEntriesWithoutContent(con, syncInfo.UpdatesWOContent)
	}
	elapsed := time.Since(updateDBStart)
	fmt.Printf("Updates to DB took: %s\n", elapsed)
}

// orchestrateSync sets up channels and workgroups to balance the workload of the syncs subprocesses
// and triggers the producer workgroup to initialize the top level sync scan and produce jobs for the other workers
func orchestrateSync(startPath string, indexedEntries map[string]data.EntryHeader, config *data.Config, syncInfo *data.SyncInfo) error {
	deletionJobs := make(chan data.DeletionJob, deletionJobBufferSize)
	scanJobs := make(chan data.EntryHeader, scanJobBufferSize)
	newDirJobs := make(chan string, newDirJobBufferSize)
	readJobs := make(chan data.SyncJob, readJobBufferSize)

	var deletionProducerWG sync.WaitGroup
	var deletionWG sync.WaitGroup
	var producerWG sync.WaitGroup
	var scannerWG sync.WaitGroup
	var readerWG sync.WaitGroup

	deletionWG.Add(deletionWorkers)
	for range deletionWorkers {
		go deletionWorker(deletionJobs, syncInfo, &deletionWG)
	}
	deletionProducerWG.Add(1)
	go produceDeletionJobs(deletionJobs, indexedEntries, &deletionProducerWG)

	scannerWG.Add(entryScanners)
	for range entryScanners {
		go scanWorker(scanJobs, readJobs, indexedEntries, &scannerWG, config)
	}

	scannerWG.Add(newDirWorkers)
	for range newDirWorkers {
		go newDirWorker(newDirJobs, readJobs, &scannerWG, indexedEntries, config)
	}

	readerWG.Add(entryReaders)
	for range entryReaders {
		go readWorker(readJobs, syncInfo, &readerWG, config)
	}

	producerWG.Add(1)
	go traverseDirectories(scanJobs, newDirJobs, readJobs, startPath, indexedEntries, &producerWG, config)

	producerWG.Wait()
	close(scanJobs)
	close(newDirJobs)

	scannerWG.Wait()
	close(readJobs)

	readerWG.Wait()
	deletionWG.Wait()

	return nil
}

// manageSync handles the sync mainloop,
// collects the program config and current index to identify deletions/changes/additions against
// decides the sync scope based on the config
// waits for sync completion before triggering index updates
func manageSync(isSyncActive *bool, syncChan chan<- struct{}) error {
	defer close(syncChan)

	scanCount := 10
	for *isSyncActive && scanCount > 0 {
		startTime := time.Now()
		config, err := config.GetConfig()
		if err != nil {
			fmt.Println(err)
		}
		var startPath string

		if scanCount%config.LargeSyncFrequenzy == 0 {
			startPath = config.LargeSyncPath
		} else {
			startPath = config.QuickSyncPath
		}

		con, err := db.CreateConnection()
		if err != nil {
			return err
		}
		defer func(con *sql.DB) {
			err = db.CloseConnection(con)
			if err != nil {
				fmt.Println(err)
			}
		}(con)

		indexedEntries, err := data.GetIndexedEntries(con)
		if err != nil {
			fmt.Println(err)
		}

		syncInfo := data.SyncInfo{}
		err = orchestrateSync(startPath, indexedEntries, &config, &syncInfo)
		if err != nil {
			return err
		}
		elapsed := time.Since(startTime)
		fmt.Printf("Scan of %s completed in: %s\n", startPath, elapsed)

		updateAfterSync(&syncInfo, con)

		time.Sleep(1 * time.Second)

		if scanCount == 1 {
			scanCount = 10
		} else {
			scanCount--
		}

	}
	return nil
}
