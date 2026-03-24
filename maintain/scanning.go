package maintain

import (
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

func scanWorker(scanJobs <-chan data.EntryHeader, readJobs chan<- data.SyncJob, indexedEntries map[string]data.EntryHeader, wg *sync.WaitGroup, config *data.Config) {
	defer wg.Done()
	for job := range scanJobs {
		err := scanUpdatedDir(readJobs, job.Path, indexedEntries, config)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// scanUpdatedDir scans the direct entries (does not crawl the file system tree to lower levels) of a directory that has been identified as changed
// categorizes the entries and produce parameterized readjobs with the entry stat and stat_t details.
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
		uniqueKey := strconv.FormatUint(entryStatT.Dev, 10) + strconv.FormatUint(entryStatT.Ino, 10) + filePath

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

// newDirWorker is responsible for traversing newly created directories and categorize new file system entries to produce readjobs of the entries
func newDirWorker(newDirJobs <-chan string, readJobs chan<- data.SyncJob, wg *sync.WaitGroup, indexedEntries map[string]data.EntryHeader, config *data.Config) {
	defer wg.Done()
	for path := range newDirJobs {
		err := traverseNewDir(readJobs, path, indexedEntries, config)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// traverseNewDir crawls a newly created directory and it's file system tree
// categorizes the entries and produces parameterized readjobs with the entries stat and stat_t details.
func traverseNewDir(
	readJobs chan<- data.SyncJob, 
	startPath string, 
	indexedEntries map[string]data.EntryHeader, 
	config *data.Config,
) error {
	err := filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && slices.Contains(config.ExcludedEntries, filepath.Base(path)) {
			return filepath.SkipDir
		}

		entryStat, err := os.Lstat(path)
		if err != nil {
			return err
		}

		entryStatT := entryStat.Sys().(*syscall.Stat_t)
		uniqueKey := strconv.FormatUint(entryStatT.Dev, 10) + strconv.FormatUint(entryStatT.Ino, 10) + path
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

		entryStat, err := os.Lstat(path)
		if err != nil {
			return nil
		}

		statT := entryStat.Sys().(*syscall.Stat_t)
		uniqueKey := strconv.FormatUint(statT.Dev, 10) + strconv.FormatUint(statT.Ino, 10) + path
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
