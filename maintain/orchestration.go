package maintain

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"igloo/data"
	"igloo/db"
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

func orchestrateScan(startPath string, config *data.Config) error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dbPath := filepath.Join(homePath, ".igloo", "igloo.db")

	con, err := db.CreateConnection(dbPath)
	if err != nil {
		return err
	}
	defer func(con *sql.DB) {
		err = db.CloseConnection(con)
		if err != nil {
			fmt.Println(err)
		}
	}(con)

	inodeMappedEntries, err := data.GetInodeMappedEntries(con)
	if err != nil {
		fmt.Println(err)
	}

	deletionJobs := make(chan string, deletionJobBufferSize)
	scanJobs := make(chan data.InodeHeader, scanJobBufferSize)
	newDirJobs := make(chan string, newDirJobBufferSize)
	readJobs := make(chan data.SyncJob, readJobBufferSize)

	var deletionProdWG sync.WaitGroup
	var deletionWG sync.WaitGroup
	var producerWG sync.WaitGroup
	var scannerWG sync.WaitGroup
	var readerWG sync.WaitGroup

	deletionWG.Add(deletionWorkers)
	for i := 0; i < deletionWorkers; i++ {
		go deletionWorker(deletionJobs, con, &deletionWG)
	}

	deletionProdWG.Add(1)
	traverseIndexedEntries(deletionJobs, inodeMappedEntries, &deletionProdWG)

	scannerWG.Add(entryScanners)
	for i := 0; i < entryScanners; i += 1 {
		go scanWorker(scanJobs, readJobs, inodeMappedEntries, &scannerWG, config)
	}

	scannerWG.Add(newDirWorkers)
	for i := 0; i < newDirWorkers; i += 1 {
		go newDirWorker(newDirJobs, readJobs, con, &scannerWG, config)
	}

	readerWG.Add(entryReaders)
	for i := 0; i < entryReaders; i += 1 {
		go readWorker(readJobs, con, &readerWG, config)
	}

	producerWG.Add(1)
	go traverseDirectories(scanJobs, newDirJobs, readJobs, startPath, inodeMappedEntries, &producerWG, config)

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
