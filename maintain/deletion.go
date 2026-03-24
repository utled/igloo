package maintain

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"igloo/data"
)


func checkDelete(entry data.DeletionJob, syncInfo *data.SyncInfo) error {
	if entryStat, err := os.Lstat(entry.Path); err != nil {
		if os.IsNotExist(err) {
			syncInfo.Mu.Lock()
			defer syncInfo.Mu.Unlock()
			syncInfo.Deletions = append(syncInfo.Deletions, &entry)
		} else {
			statT := entryStat.Sys().(*syscall.Stat_t)
			if entry.DevID != statT.Dev || entry.Inode != statT.Ino {
				syncInfo.Deletions = append(syncInfo.Deletions, &entry)
			}
		}
	}
	return nil
}

func produceDeletionJobs(deletionJobs chan<- data.DeletionJob, indexedEntries map[string]data.EntryHeader, wg *sync.WaitGroup) error {
	defer wg.Done()
	defer close(deletionJobs)

	for uniqueKey, entry := range indexedEntries {
		deletionJobs <- data.DeletionJob{UniqueKey: uniqueKey, DevID: entry.DevID, Inode: entry.Inode, Path: entry.Path}
	}
	return nil
}

func deleteAfterScan(con *sql.DB, syncInfo *data.SyncInfo) error {
	startTime := time.Now()
	countOfDeletions := len(syncInfo.Deletions)
	//fmt.Printf("Starting DB deletions for: %d entries\n", countOfDeletions)
	if countOfDeletions > 0 {
		data.DeleteEntries(con, syncInfo.Deletions)
	}
	
	elsapsed := time.Since(startTime)
	fmt.Println("DB deletions took: ", elsapsed)
	return nil
}
