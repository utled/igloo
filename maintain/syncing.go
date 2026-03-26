package maintain

import (
	"database/sql"
	"fmt"
	"log/slog"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"igloo/config"
	"igloo/data"
	"igloo/db"
	"igloo/logging"
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
	slog.Debug(fmt.Sprintf("Updating DB for: %d Deletions - %d New entries - %d Updates with content - %d Updates without content",
		countOfDeletions,
		countOfNewEntries,
		countOfUpdatesWContent,
		countOfUpdatesWOContent,
	))
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
	slog.Debug(fmt.Sprintf("Updates to DB took: %s", elapsed), "call", "maintain.updateAfterSync()")
}

// startSync sets up channels and workgroups to balance the workload of the syncs subprocesses
// and triggers the producer workgroup to initialize the top level sync scan and produce jobs for the other workers
func startSync(startPath string, indexedEntries map[string]data.EntryHeader, config *data.Config, syncInfo *data.SyncInfo) {
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
			return fmt.Errorf("maintain.orchestrateSync() -> db.CreateConnection() %w", err)
		}
		defer func(con *sql.DB) {
			err = db.CloseConnection(con)
			if err != nil {
				slog.Error("failed to close db connection", "call", "db.CloseConnection()", "err", err)
			}
		}(con)

		indexedEntries, err := data.GetIndexedEntries(con)
		if err != nil {
			return fmt.Errorf("maintain.orchestrateSync -> data.GetIndexedEntries() %w", err)
		}

		config, err := config.GetConfig()
		if err != nil {
			return fmt.Errorf("maintain.orchestrateSync -> config.GetConfig() %w", err)
		}
		logging.ChangeLogLevel(config.LogLevel)

		syncInfo := data.SyncInfo{}
		startSync(config.SyncPath, indexedEntries, &config, &syncInfo)
		indexedEntries = nil
		elapsed := time.Since(startTime)
		slog.Debug(fmt.Sprintf("Scan duration for %s: %s\n", config.SyncPath, elapsed))

		updateAfterSync(&syncInfo, con)
		syncInfo = data.SyncInfo{}
		runtime.GC()
		debug.FreeOSMemory()

		time.Sleep(time.Duration(config.WaitBetweenSyncs) * time.Second)
	}
	return nil
}
