// Package initial manages the full initial file system scan and indexing
package initial

import (
	"fmt"
	"igloo/config"
	"igloo/data"
	"log"
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
	theWorks := data.CollectedInfo{}

	fileReadJobs := make(chan data.ReadJob, fileJobBufferSize)
	dirReadJobs := make(chan data.ReadJob, directoryJobBufferSize)

	var wg sync.WaitGroup
	totalWorkers := 1 + directoryWorkers + fileWorkers
	wg.Add(totalWorkers)

	config, err := config.GetConfig()
	if err != nil {
		fmt.Println(err)
	}
	stat, err := os.Lstat(config.SyncPath)
	if err != nil {
		fmt.Println(err)
	}
	if !stat.IsDir() {
		log.Fatal("Starting path must be a directory")
	}
	readDir(config.SyncPath, &stat, &theWorks, true)

	for i := 0; i < directoryWorkers; i += 1 {
		go dirWorker(dirReadJobs, &wg, &theWorks)
	}

	for i := 0; i < fileWorkers; i += 1 {
		go fileWorker(fileReadJobs, &wg, &theWorks, &config)
	}

	go traverseDirectory(config.SyncPath, dirReadJobs, fileReadJobs, &wg, &theWorks, &config)

	wg.Wait()
	end := time.Now()
	elapsed := end.Sub(start)

	fmt.Printf("Full scan took %s\n", elapsed)

	theWorks.Mu.Lock()
	theWorks.ScanStart = start
	theWorks.ScanEnd = end
	theWorks.ScanDuration = elapsed
	theWorks.Mu.Unlock()

	writeStart := time.Now()
	err = writeFullIndex(&theWorks)
	if err != nil {
		fmt.Println(err)
	}
	writeElapsed := time.Since(writeStart)
	fmt.Printf("Full dbWrite took %s\n", writeElapsed)
	theWorks = data.CollectedInfo{}
	runtime.GC()
	debug.FreeOSMemory()
}
