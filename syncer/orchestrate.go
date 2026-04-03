package syncer

import (
	"database/sql"
	"fmt"
	"log/slog"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"igloo/config"
	"igloo/db"
	"igloo/indexer"
	"igloo/logger"
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

// startSync sets up channels and workgroups to balance the workload of the syncs subprocesses
// and triggers the producer workgroup to initialize the top level sync scan and produce jobs for the other workers
func startSync(startPath string, indexedEntries map[string]entryHeader, syncDetails *syncCollection) {
	deletionJobs := make(chan deletionJob, deletionJobBufferSize)
	scanJobs := make(chan entryHeader, scanJobBufferSize)
	newDirJobs := make(chan string, newDirJobBufferSize)
	readJobs := make(chan syncJob, readJobBufferSize)

	var deletionProducerWG sync.WaitGroup
	var deletionWG sync.WaitGroup
	var producerWG sync.WaitGroup
	var scannerWG sync.WaitGroup
	var readerWG sync.WaitGroup

	deletionWG.Add(deletionWorkers)
	for range deletionWorkers {
		go deletionWorker(deletionJobs, syncDetails, &deletionWG)
	}
	deletionProducerWG.Add(1)
	go produceDeletionJobs(deletionJobs, indexedEntries, &deletionProducerWG)

	scannerWG.Add(entryScanners)
	for range entryScanners {
		go scanWorker(scanJobs, readJobs, indexedEntries, &scannerWG)
	}

	scannerWG.Add(newDirWorkers)
	for range newDirWorkers {
		go newDirWorker(newDirJobs, readJobs, &scannerWG, indexedEntries)
	}

	readerWG.Add(entryReaders)
	for range entryReaders {
		go readWorker(readJobs, syncDetails, &readerWG)
	}

	producerWG.Add(1)
	go traverseDirectories(scanJobs, newDirJobs, readJobs, startPath, indexedEntries, &producerWG)

	producerWG.Wait()
	close(scanJobs)
	close(newDirJobs)

	scannerWG.Wait()
	close(readJobs)

	readerWG.Wait()
	deletionWG.Wait()
}

type entryHeader struct {
	devID              uint64
	inode              uint64
	path               string
	modificationTime   time.Time
	metaDataChangeTime time.Time
}

// getIndexedEntries fetches all indexed entries from db with the relevant details for the sync process to identify changes on the entries
// dev_id + inode + path are combined as the primary key/unique identifier of a file system entry
func getIndexedEntries(con *sql.DB) (indexedEntries map[string]entryHeader, err error) {
	indexedEntries = make(map[string]entryHeader)
	var query string
	var response *sql.Rows
	query = `select dev_id, inode, path, modification_time, metadata_change_time 
				from entries
				order by inode;`
	response, err = con.Query(query)
	if err != nil {
		return indexedEntries, fmt.Errorf("syncer.getIndexedEntries() -> con.Query() %w", err)
	}

	for response.Next() {
		var details entryHeader
		err = response.Scan(
			&details.devID,
			&details.inode,
			&details.path,
			&details.modificationTime,
			&details.metaDataChangeTime,
		)
		if err != nil {
			return indexedEntries, fmt.Errorf("syncer.etIndexedEntries() -> response.Scan() %w", err)
		}
		uniqueKey := strconv.FormatUint(details.devID, 10) + strconv.FormatUint(details.inode, 10) + details.path
		indexedEntries[uniqueKey] = details
	}
	if err = response.Err(); err != nil {
		return indexedEntries, fmt.Errorf("syncer.etIndexedEntries() -> response.Next() %w", err)
	}

	return indexedEntries, nil
}

type syncCollection struct {
	deletions        []*deletionJob
	newEntries       []*indexer.EntryCollection
	updatesWContent  []*indexer.EntryCollection
	updatesWOContent []*indexer.EntryCollection
	mu               sync.Mutex
}

// orchestrateSync handles the sync mainloop,
// collects the program config and current index to identify deletions/changes/additions against
// decides the sync scope based on the config
// waits for sync completion before triggering index updates
func orchestrateSync(isSyncActive *bool, syncChan chan<- struct{}) error {
	defer close(syncChan)

	for *isSyncActive {
		requiresRefresh, err := config.CheckUpdate()
		if err != nil {
			return fmt.Errorf("syncer.orchestrateSync -> utils.GetConfig() %w", err)
		}
		logger.CheckUpdateLogLevel()
		if requiresRefresh {
			indexer.StartFullScan()
		}

		startTime := time.Now()

		con, err := db.CreateConnection()
		if err != nil {
			return fmt.Errorf("syncer.orchestrateSync() -> db.CreateConnection() %w", err)
		}
		defer func(con *sql.DB) {
			err = db.CloseConnection(con)
			if err != nil {
				slog.Error("failed to close db connection", "call", "db.CloseConnection()", "err", err)
			}
		}(con)

		indexedEntries, err := getIndexedEntries(con)
		if err != nil {
			return fmt.Errorf("syncer.orchestrateSync -> data.GetIndexedEntries() %w", err)
		}

		syncDetails := syncCollection{}
		startSync(config.Details.SyncPath, indexedEntries, &syncDetails)
		indexedEntries = nil
		elapsed := time.Since(startTime)
		slog.Debug(fmt.Sprintf("Scan duration for %s: %s\n", config.Details.SyncPath, elapsed))

		updateAfterSync(&syncDetails, con)
		syncDetails = syncCollection{}
		runtime.GC()
		debug.FreeOSMemory()

		time.Sleep(time.Duration(config.Details.WaitBetweenSyncs) * time.Second)
	}
	return nil
}
