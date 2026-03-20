package maintain

import (
	"fmt"
	"igloo/data"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"syscall"
	"time"
)

func scanUpdatedDir(readJobs chan<- data.SyncJob, dirPath string, uniqueIndexedEntries map[string]data.EntryHeader, config *data.Config) error {
	fileSysEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to list entries in directory: %s\n%w", dirPath, err)
	}

	for _, entry := range fileSysEntries {
		filePath := filepath.Join(dirPath, entry.Name())

		entryStat, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		if entryStat.IsDir() && slices.Contains(config.ExcludedEntries, filepath.Base(filePath)) {
			continue
		}

		entryStatT := entryStat.Sys().(*syscall.Stat_t)
		entryMtim := time.Unix(entryStatT.Mtim.Sec, entryStatT.Mtim.Nsec)
		uniqueKey := strconv.Itoa(int(entryStatT.Dev)) + strconv.Itoa(int(entryStatT.Ino)) + filePath
		if inode, ok := uniqueIndexedEntries[uniqueKey]; !ok {
			if !entryStat.IsDir() {
				syncJob := data.SyncJob{Path: filePath, IsIndexed: false, IsContentChange: true}
				readJobs <- syncJob
			}
		} else {
			if !entryMtim.Equal(inode.ModificationTime) {
				syncJob := data.SyncJob{Path: filePath, IsIndexed: true, IsContentChange: !entry.IsDir()}
				readJobs <- syncJob
			} else {
				syncJob := data.SyncJob{Path: filePath, IsIndexed: true, IsContentChange: false}
				readJobs <- syncJob
			}
		}
	}

	return nil
}
