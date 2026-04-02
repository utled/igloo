package initial

import (
	"database/sql"
	"igloo/data"
	"log/slog"
	"sync"
)


func batchWorker(batchJobs <-chan *data.EntryCollection, countChan chan<- int, batchSize int, wg *sync.WaitGroup, con *sql.DB) {
	defer wg.Done()

	var batchedEntries []*data.EntryCollection
	var batchCount int
	for job := range batchJobs {
		batchedEntries = append(batchedEntries, job)
		batchCount++
		if batchCount == batchSize {
			err := data.WriteFullEntries(con, batchedEntries)
			if err != nil {
				slog.Error("on looped batch", "call", "data.WriteFullEntries()", "err", err)
			}
			countChan <- batchCount
			batchedEntries = nil
			batchCount = 0
		}
	}
	err := data.WriteFullEntries(con, batchedEntries)
	if err != nil {
		slog.Error("on final batch", "call", "data.WriteFullEntries()", "err", err)
	}
	countChan <- batchCount
}
