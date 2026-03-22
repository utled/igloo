package maintain

import (
	"os"
	"sync"

	"igloo/data"
)

func checkDelete(entry data.DeletionJob, syncInfo *data.SyncInfo) error {
	if /*entryStat*/_, err := os.Lstat(entry.Path); err != nil {
		if os.IsNotExist(err) {
			syncInfo.Mu.Lock()
			defer syncInfo.Mu.Unlock()
			syncInfo.Deletions = append(syncInfo.Deletions, &entry)
		}
	} /*else {
		statT := entryStat.Sys().(*syscall.Stat_t)
		syncInfo.Mu.Lock()
		defer syncInfo.Mu.Unlock()
		if entry.DevID != statT.Dev || entry.Inode != statT.Ino {
			syncInfo.Deletions = append(syncInfo.Deletions, &entry)
		}
	}
	*/

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
