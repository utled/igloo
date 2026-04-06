package syncer

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
)

type deletionJob struct {
	uniqueKey string
	devID     uint64
	inode     uint64
	path      string
}

func deleteEntries(con *sql.DB, deletionEntries []*deletionJob) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("syncer.deleteEntries() -> con.Begin() %w", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare(`delete from entries where dev_id = ? and inode = ? and path = ?`)
	if err != nil {
		return fmt.Errorf("syncer.deleteEntries() -> transaction.Prepare() %w", err)
	}
	defer statement.Close()

	for _, entry := range deletionEntries {
		_, err := statement.Exec(
			entry.devID,
			entry.inode,
			entry.path,
		)
		if err != nil {
			return fmt.Errorf("syncer.deleteEntries() -> statement.Exec() for entry key: %s %w", entry.uniqueKey, err)
		}
	}

	return transaction.Commit()
}

func deletionWorker(delJobs <-chan deletionJob, syncDetails *syncCollection, wg *sync.WaitGroup) {
	defer wg.Done()
	counter := 0
	for entry := range delJobs {
		checkDelete(entry, syncDetails)
		counter++
		if counter == 20 {
			time.Sleep(1 * time.Millisecond)
			counter = 0
		}
	}
}

// checkDelete checks if a file system entry still exist on disk and collects no longer existing entries to to be deleted at the end of the syncloop
func checkDelete(entry deletionJob, syncDetails *syncCollection) {
	if entryStat, err := os.Lstat(entry.path); err != nil {
		if os.IsNotExist(err) {
			syncDetails.mu.Lock()
			defer syncDetails.mu.Unlock()
			syncDetails.deletions = append(syncDetails.deletions, &entry)
		} else {
			statT := entryStat.Sys().(*syscall.Stat_t)
			if entry.devID != statT.Dev || entry.inode != statT.Ino {
				syncDetails.mu.Lock()
				defer syncDetails.mu.Unlock()
				syncDetails.deletions = append(syncDetails.deletions, &entry)
			}
		}
	}
}

func produceDeletionJobs(deletionJobs chan<- deletionJob, indexedEntries map[string]entryHeader, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(deletionJobs)

	for uniqueKey, entry := range indexedEntries {
		deletionJobs <- deletionJob{uniqueKey: uniqueKey, devID: entry.devID, inode: entry.inode, path: entry.path}
	}
}
