// Package initial manages the full initial file system scan and indexing
package initial

import (
	"fmt"
	"igloo/utils"
	"igloo/data"
	"log/slog"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

const (
	directoryWorkers       = 10
	fileWorkers            = 30
	directoryJobBufferSize = 100
	fileJobBufferSize      = 400
)

func StartInitialScan() {
	start := time.Now()

	err := utils.GetConfig()
	if err != nil {
		slog.Error("", "call", "utils.GetConfig()", "err", err)
	}
	utils.CheckUpdateLogLevel()
	syncPath := utils.Config.SyncPath
	
	theWorks := data.CollectedInfo{}

	fileReadJobs := make(chan data.ReadJob, fileJobBufferSize)
	dirReadJobs := make(chan data.ReadJob, directoryJobBufferSize)

	var wg sync.WaitGroup
	totalWorkers := 1 + directoryWorkers + fileWorkers
	wg.Add(totalWorkers)

	stat, err := os.Lstat(syncPath)
	if err != nil {
		slog.Error("failed to run Lstat on initial sync path", "call", "os.Lstat()", "err", err)
	}
	if !stat.IsDir() {
		slog.Error("initial sync path is not a directory", "call", "!stat.IsDir()")
	}
	readDir(syncPath, &stat, &theWorks, true)

	for i := 0; i < directoryWorkers; i += 1 {
		go dirWorker(dirReadJobs, &wg, &theWorks)
	}

	for i := 0; i < fileWorkers; i += 1 {
		go fileWorker(fileReadJobs, &wg, &theWorks)
	}
	go traverseDirectory(syncPath, dirReadJobs, fileReadJobs, &wg)

	wg.Wait()
	end := time.Now()
	elapsed := end.Sub(start)

	slog.Debug(fmt.Sprintf("initial scan duration: %s", elapsed))
	
	writeStart := time.Now()
	err = writeFullIndex(&theWorks)
	if err != nil {
		slog.Error("failed to write initial index to DB", "call", "initial.writeFullIndex()", "err", err)
	}
	writeElapsed := time.Since(writeStart)
	slog.Debug(fmt.Sprintf("initial db write duration: %s\n", writeElapsed))
	
	countOfEntries := len(theWorks.EntryDetails)
	utils.Notify(fmt.Sprintf("Full file system scan completed\n%d entries have been indexed", countOfEntries), false)

	theWorks = data.CollectedInfo{}
	runtime.GC()
	debug.FreeOSMemory()
}
