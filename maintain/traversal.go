package maintain

import (
	"database/sql"
	"fmt"
	"igloo/data"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"time"
)

func traverseNewDir(readJobs chan<- data.SyncJob, startPath string, config *data.Config, con *sql.DB) error {
	indexedEntries, err := data.GetIndexedEntries(con)
	if err != nil {
		return err
	}
	err = filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && slices.Contains(config.ExcludedEntries, filepath.Base(path)) {
			return filepath.SkipDir
		}

		entryStat, err := os.Stat(path)
		if err != nil {
			return err
		}

		entryStatT := entryStat.Sys().(*syscall.Stat_t)
		uniqueKey := strconv.Itoa(int(entryStatT.Dev)) + strconv.Itoa(int(entryStatT.Ino)) + path
		indexedEntry, isIndexed := indexedEntries[uniqueKey]
		isContentChange := false

		if !entryStat.IsDir() {
			if !isIndexed {
				isContentChange = true
			} else {
				entryMtim := time.Unix(entryStatT.Mtim.Sec, entryStatT.Mtim.Nsec)				
				isContentChange = !entryMtim.Equal(indexedEntry.ModificationTime)
			}
		}

		readJobs <- data.SyncJob{
			Path: path,
			IsIndexed: isIndexed, 
			IsContentChange: isContentChange,
			Stat: &entryStat,
			StatT: *entryStatT,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to traverse new directory:%v", err)
	}

	return nil
}

// traverseDirectories crawls the directory tree from top level and checks for mTim/cTim changes to identify if anything has changed within the directory
// if changes are identified, it produces a readJob to record the metadata of the direcotry itself and
// a scanJob to read the contents of the directory to identify what has changed at lower levels
// if the directory is not previously indexed (with the same unique key, i.e. dev_id+inode+path), it produces a newDirJob to perform a full scan of the directory tree below it
func traverseDirectories(
	scanJobs chan<- data.EntryHeader,
	newDirJobs chan<- string,
	readJobs chan<- data.SyncJob,
	startPath string,
	indexedEntries map[string]data.EntryHeader,
	wg *sync.WaitGroup,
	config *data.Config,
) {
	defer wg.Done()

	err := filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && slices.Contains(config.ExcludedEntries, filepath.Base(path)) {
			return filepath.SkipDir
		}

		entryStat, err := os.Stat(path)
		if err != nil {
			return nil
		}

		statT := entryStat.Sys().(*syscall.Stat_t)
		uniqueKey := strconv.Itoa(int(statT.Dev)) + strconv.Itoa(int(statT.Ino)) + path
		if d.IsDir() {
			if indexedEntry, isIndexed := indexedEntries[uniqueKey]; !isIndexed {
				newDirJobs <- path
				return filepath.SkipDir
			} else {
				mTim := time.Unix(statT.Mtim.Sec, statT.Mtim.Nsec)
				cTim := time.Unix(statT.Ctim.Sec, statT.Ctim.Nsec)
				if !indexedEntry.ModificationTime.Equal(mTim) || !indexedEntry.MetaDataChangeTime.Equal(cTim) {
					scanJobs <- indexedEntry
					readJobs <- data.SyncJob{
						Path: path, 
						IsIndexed: true, 
						IsContentChange: false, 
						Stat: &entryStat, 
						StatT: *statT,
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Fatal error during directory traversal: %v", err)
	}
}
