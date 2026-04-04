package indexer

import (
	"database/sql"
	"log/slog"
	"sync"
)

// batchWorker groups entries for indexing into groups with a size defined by const batchSize and sends the batches to the db write function in a new goroutine
// it also keeps track of how many entries it has collected and send the info back through the countChan for total count in the indexer main process
// the batching functionality was introduced to reduce the runtime memory consumption due to the full indexing process managing potentially very large volumes of data
// by batching the writing, the indexer doesn't need to keep the whole file system in memory during the process
func batchWorker(batchJobs <-chan *EntryCollection, countChan chan<- int, batchSize int, batchWH *sync.WaitGroup, writeWG *sync.WaitGroup, con *sql.DB) {
	defer batchWH.Done()

	batchedEntries := make([]*EntryCollection, 0, batchSize)
	var batchCount int
	for job := range batchJobs {
		batchedEntries = append(batchedEntries, job)
		batchCount++
		if batchCount == batchSize {
			writeWG.Add(1)
			go func(batchedEntries []*EntryCollection) {
				err := WriteFullEntries(con, batchedEntries, writeWG)
				if err != nil {
					slog.Error("on looped batch", "call", "data.WriteFullEntries()", "err", err)
				}
			}(batchedEntries)
			countChan <- batchCount
			batchedEntries = nil
			batchCount = 0
		}
	}
	writeWG.Add(1)
	go func(batchedEntries []*EntryCollection) {
		err := WriteFullEntries(con, batchedEntries, writeWG)
		if err != nil {
			slog.Error("on final batch", "call", "data.WriteFullEntries()", "err", err)
		}
	}(batchedEntries)
	countChan <- batchCount
}
