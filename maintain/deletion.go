package maintain

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	"igloo/data"
	"igloo/db"
)

const (
	deletionWorkers       = 20
	deletionJobBufferSize = 100
)

func deletionWorker(delJobs <-chan data.DeletionJob, deletions *data.DeletionInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range delJobs {
		err := checkDelete(path, deletions)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func checkDelete(entry data.DeletionJob, deletions *data.DeletionInfo) error {
	if _, err := os.Lstat(entry.Path); err != nil {
		if os.IsNotExist(err) {
			deletions.Mu.Lock()
			defer deletions.Mu.Unlock()
			deletions.Deletions = append(deletions.Deletions, &entry)
		}
	}

	return nil
}

func traverseIndexedEntries(deletionJobs chan<- data.DeletionJob, indexedEntries map[string]data.EntryHeader, wg *sync.WaitGroup) error {
	defer wg.Done()
	defer close(deletionJobs)

	for uniqueKey, entry := range indexedEntries {
		deletionJobs <- data.DeletionJob{UniqueKey: uniqueKey, DevID: entry.DevID, Inode: entry.Inode, Path: entry.Path}
	}
	return nil
}

func manageDeletions(isSyncActive *bool, deletionChan chan<- struct{}) error {
	defer close(deletionChan)
	for *isSyncActive {
		con, err := db.CreateConnection()
		if err != nil {
			return err
		}
		defer func(con *sql.DB) {
			err = db.CloseConnection(con)
			if err != nil {
				fmt.Println(err)
			}
		}(con)

		indexedEntries, err := data.GetIndexedEntries(con)
		if err != nil {
			fmt.Println(err)
		}

		deletionJobs := make(chan data.DeletionJob, deletionJobBufferSize)
		var deletionProdWG sync.WaitGroup
		var deletionWG sync.WaitGroup

		deletions := data.DeletionInfo{}

		deletionWG.Add(deletionWorkers)
		for range deletionWorkers {
			go deletionWorker(deletionJobs, &deletions, &deletionWG)
		}

		deletionProdWG.Add(1)
		traverseIndexedEntries(deletionJobs, indexedEntries, &deletionProdWG)

		deletionProdWG.Wait()
		deletionWG.Wait()
	}

	return nil
}
