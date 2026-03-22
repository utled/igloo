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

func scanUpdatedDir(readJobs chan<- data.SyncJob, dirPath string, indexedEntries map[string]data.EntryHeader, config *data.Config) error {
	fileSysEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to list entries in directory: %s\n%w", dirPath, err)
	}

	for _, entry := range fileSysEntries {
		filePath := filepath.Join(dirPath, entry.Name())

		entryStat, err := os.Lstat(filePath)
		if err != nil {
			return err
		}

		isDir := entryStat.IsDir()

		if isDir && slices.Contains(config.ExcludedEntries, filepath.Base(filePath)) {
			continue
		}

		entryStatT := entryStat.Sys().(*syscall.Stat_t)
		entryMtim := time.Unix(entryStatT.Mtim.Sec, entryStatT.Mtim.Nsec)
		uniqueKey := strconv.Itoa(int(entryStatT.Dev)) + strconv.Itoa(int(entryStatT.Ino)) + filePath

		indexedEntry, isIndexed := indexedEntries[uniqueKey]
		isContentChange := false
		if !isDir {
			if !isIndexed || entryMtim.Equal(indexedEntry.ModificationTime) {
				isContentChange = true
			}
		}
		
		readJobs <- data.SyncJob{
			Path: filePath,
			IsIndexed: isIndexed,
			IsContentChange: isContentChange,
			Stat: &entryStat,
			StatT: *entryStatT,
		}
	}

	return nil
}
