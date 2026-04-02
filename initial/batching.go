package initial

import (
	"database/sql"
	"log/slog"
	"sync"

	"igloo/data"
)

func batchWorker(batchJobs <-chan *data.EntryCollection, countChan chan<- int, batchSize int, batchWH *sync.WaitGroup, writeWG *sync.WaitGroup, con *sql.DB) {
	defer batchWH.Done()

	var batchedEntries []*data.EntryCollection
	var batchCount int
	for job := range batchJobs {
		batchedEntries = append(batchedEntries, job)
		batchCount++
		if batchCount == batchSize {
			writeWG.Add(1)
			go func(batchedEntries []*data.EntryCollection) {
				err := data.WriteFullEntries(con, batchedEntries, writeWG)
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
	go func(batchedEntries []*data.EntryCollection) {
		err := data.WriteFullEntries(con, batchedEntries, writeWG)
		if err != nil {
			slog.Error("on final batch", "call", "data.WriteFullEntries()", "err", err)
		}
	}(batchedEntries)
	countChan <- batchCount
}
