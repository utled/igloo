// Package initial manages the full initial file system scan and indexing
package initial

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"igloo/data"
	"igloo/db"
	"igloo/utils"
)

const (
	readWorkers        = 4
	batchWorkers       = 2
	readJobBufferSize  = 8
	batchJobBufferSize = 3000
	batchSize          = 20000
)

func clearIndex(con *sql.DB) error {
	err := data.ClearExistingData(con)
	if err != nil {
		return fmt.Errorf("initial.clearIndex() -> data.ClearExistingData() %w", err)
	}

	return nil
}

func StartInitialScan() error {
	start := time.Now()

	err := utils.GetConfig()
	if err != nil {
		return fmt.Errorf("initial.StartInitialScan() -> utils.GetConfig() %w", err)
	}
	utils.CheckUpdateLogLevel()

	syncPath := utils.Config.SyncPath
	stat, err := os.Lstat(syncPath)
	if err != nil {
		return fmt.Errorf("initial.StartInitialScan() -> os.Lstat() %w", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("initial.StartInitialScan() -> !stat.IsDir() %w", err)
	}

	con, err := db.CreateConnection()
	if err != nil {
		return fmt.Errorf("initial.StartInitialScan() -> db.CreateConnection() %w", err)
	}
	defer func(con *sql.DB) {
		err = db.CloseConnection(con)
		if err != nil {
			slog.Error("failed to close db connection", "call", "db.CloseConnection()", "err", err)
		}
	}(con)

	err = clearIndex(con)
	if err != nil {
		return fmt.Errorf("initial.StartInitialScan() -> iniial.clearIndex() %w", err)
	}

	
	var indexCount int
	countChan := make(chan int)
	go func(countChannel <-chan int) {
		for value := range countChannel {
			indexCount += value
		}
	}(countChan)

	batchJobs := make(chan *data.EntryCollection, batchJobBufferSize)
	var batchWG sync.WaitGroup
	var writeWG sync.WaitGroup
	batchWG.Add(batchWorkers)
	for i := 0; i < batchWorkers; i += 1 {
		go batchWorker(batchJobs, countChan, batchSize, &batchWG, &writeWG, con)
	}

	readJobs := make(chan data.ReadJob, readJobBufferSize)
	var collectorWG sync.WaitGroup
	collectorWorkers := 1 + readWorkers
	collectorWG.Add(collectorWorkers)

	for i := 0; i < readWorkers; i += 1 {
		go readWorker(readJobs, batchJobs, &collectorWG)
	}

	go traverseDirectory(syncPath, readJobs, &collectorWG)

	collectorWG.Wait()
	close(batchJobs)

	batchWG.Wait()
	close(countChan)
	writeWG.Wait()

	end := time.Now()
	elapsed := end.Sub(start)

	slog.Debug(fmt.Sprintf("full scan of %d entries in %s", indexCount, elapsed))
	utils.Notify(fmt.Sprintf("Full file system scan completed\n%d entries have been indexed\nduration: %s", indexCount, elapsed), false)

	runtime.GC()
	debug.FreeOSMemory()
	return nil
}
