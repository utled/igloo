package maintain

import (
	"database/sql"
	"igloo/data"
	"sync"
)

const (
	deletionJobBufferSize = 100
	scanJobBufferSize     = 100
	readJobBufferSize     = 500
	newDirJobBufferSize   = 100
	deletionWorkers       = 20
	entryScanners         = 20
	entryReaders          = 80
	newDirWorkers         = 20
)

func orchestrateScan(startPath string, indexedEntries map[string]data.EntryHeader, config *data.Config, syncInfo *data.SyncInfo, con *sql.DB) error {
	deletionJobs := make(chan data.DeletionJob, deletionJobBufferSize)
	scanJobs := make(chan data.EntryHeader, scanJobBufferSize)
	newDirJobs := make(chan string, newDirJobBufferSize)
	readJobs := make(chan data.SyncJob, readJobBufferSize)

	var deletionProdWG sync.WaitGroup
	var deletionWG sync.WaitGroup
	var producerWG sync.WaitGroup
	var scannerWG sync.WaitGroup
	var readerWG sync.WaitGroup

	deletionWG.Add(deletionWorkers)
	for range deletionWorkers {
		go deletionWorker(deletionJobs, syncInfo, &deletionWG)
	}

	deletionProdWG.Add(1)
	traverseIndexedEntries(deletionJobs, indexedEntries, &deletionProdWG)

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
	deletionProdWG.Wait()
	deletionWG.Wait()

	return nil
}
