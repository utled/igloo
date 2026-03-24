package maintain

import (
	"fmt"
	"os"
	"sync"
	"syscall"

	"igloo/data"
)

func deletionWorker(delJobs <-chan data.DeletionJob, syncInfo *data.SyncInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	for path := range delJobs {
		err := checkDelete(path, syncInfo)
		if err != nil {
			fmt.Println(err)
		}
	}
}

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
