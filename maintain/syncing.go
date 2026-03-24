package maintain

import (
	"database/sql"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"igloo/config"
	"igloo/data"
	"igloo/db"
)

const (
	deletionJobBufferSize = 50
	scanJobBufferSize     = 50
	readJobBufferSize     = 400
	newDirJobBufferSize   = 10
	deletionWorkers       = 20
	entryScanners         = 20
	entryReaders          = 30
	newDirWorkers         = 5
)

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

// startSync sets up channels and workgroups to balance the workload of the syncs subprocesses
// and triggers the producer workgroup to initialize the top level sync scan and produce jobs for the other workers
func startSync(startPath string, indexedEntries map[string]data.EntryHeader, config *data.Config, syncInfo *data.SyncInfo) error {
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

// orchestrateSync handles the sync mainloop,
// collects the program config and current index to identify deletions/changes/additions against
// decides the sync scope based on the config
// waits for sync completion before triggering index updates
func orchestrateSync(isSyncActive *bool, syncChan chan<- struct{}) error {
	defer close(syncChan)

	for *isSyncActive {
		startTime := time.Now()

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

		config, err := config.GetConfig()
		if err != nil {
			fmt.Println(err)
		}

		syncInfo := data.SyncInfo{}
		err = startSync(config.SyncPath, indexedEntries, &config, &syncInfo)
		if err != nil {
			return err
		}
		indexedEntries = nil
		elapsed := time.Since(startTime)
		fmt.Printf("Scan of %s completed in: %s\n", config.SyncPath, elapsed)

		updateAfterSync(&syncInfo, con)
		syncInfo = data.SyncInfo{}
		runtime.GC()
		debug.FreeOSMemory()

		time.Sleep(time.Duration(config.WaitBetweenSyncs) * time.Second)
	}
	return nil
}
