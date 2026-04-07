// Package indexer manages clearing of existing index and fresh full file system scans and indexing
package indexer

import (
	"database/sql"
	"fmt"
	"igloo/config"
	"igloo/db"
	"igloo/logger"
	"igloo/notify"
	"log/slog"
	"math/rand/v2"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

const (
	readWorkers        = 3
	batchWorkers       = 1
	readJobBufferSize  = 10
	batchJobBufferSize = 100
	batchSize          = 10_000
)

// RunFullScan is the entry point for the full file system indexer.
// It triggers a fresh read of the config file, triggers an update of the log level according to the config
// and orchestrates the resources (channels and waitgroups) and triggers the goroutines for executing the indexing process.
func RunFullScan() error {
	start := time.Now()

	err := config.Read()
	if err != nil {
		return fmt.Errorf("initial.StartInitialScan() -> utils.GetConfig() %w", err)
	}
	logger.CheckUpdateLogLevel()

	syncPath := config.Details.SyncPath
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

	err = clearExistingData(con)
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

	batchJobs := make(chan *EntryCollection, batchJobBufferSize)
	var batchWG sync.WaitGroup
	var writeWG sync.WaitGroup
	batchWG.Add(batchWorkers)
	for i := 0; i < batchWorkers; i += 1 {
		go batchWorker(batchJobs, countChan, batchSize, &batchWG, &writeWG, con)
	}

	readJobs := make(chan readJob, readJobBufferSize)
	var collectorWG sync.WaitGroup
	collectorWorkers := 1 + readWorkers
	collectorWG.Add(collectorWorkers)

	go traverseDirectory(syncPath, readJobs, &collectorWG)

	for i := 0; i < readWorkers; i += 1 {
		go readWorker(readJobs, batchJobs, &collectorWG)
		jitter := time.Duration(rand.IntN(10) * int(time.Millisecond))
		time.Sleep(5 * time.Millisecond + jitter)
	}

	collectorWG.Wait()
	close(batchJobs)

	batchWG.Wait()
	close(countChan)
	writeWG.Wait()

	end := time.Now()
	elapsed := end.Sub(start)

	slog.Info(fmt.Sprintf("full scan of %d entries in %s", indexCount, elapsed))
	notify.Send(fmt.Sprintf("Full file system scan completed\n%d entries have been indexed\nduration: %s", indexCount, elapsed), false)

	runtime.GC()
	debug.FreeOSMemory()
	return nil
}
