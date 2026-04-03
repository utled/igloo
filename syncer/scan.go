package syncer

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"time"

	"igloo/config"
)

func scanWorker(scanJobs <-chan entryHeader, readJobs chan<- syncJob, indexedEntries map[string]entryHeader, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range scanJobs {
		err := scanUpdatedDir(readJobs, job.path, indexedEntries)
		if err != nil {
			slog.Error("", "call", "syncer.scanUpdatedDir()", "err", err)
		}
	}
}

// scanUpdatedDir scans the direct entries (does not crawl the file system tree to lower levels) of a directory that has been identified as changed
// categorizes the entries and produce parameterized readjobs with the entry stat and stat_t details.
func scanUpdatedDir(readJobs chan<- syncJob, dirPath string, indexedEntries map[string]entryHeader) error {
	fileSysEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("syncer.scanUpdatedDir() -> os.ReadDir() for path %s %w", dirPath, err)
	}

	for _, entry := range fileSysEntries {
		filePath := filepath.Join(dirPath, entry.Name())

		entryStat, err := os.Lstat(filePath)
		if err != nil {
			return fmt.Errorf("syncer.scanUpdatedDir() -> os.Lstat() for path %s %w", filePath, err)
		}

		isDir := entryStat.IsDir()

		if isDir && slices.Contains(config.Details.ExcludedEntries, filepath.Base(filePath)) {
			continue
		}

		entryStatT := entryStat.Sys().(*syscall.Stat_t)
		entryMtim := time.Unix(entryStatT.Mtim.Sec, entryStatT.Mtim.Nsec)
		uniqueKey := strconv.FormatUint(entryStatT.Dev, 10) + strconv.FormatUint(entryStatT.Ino, 10) + filePath

		indexedEntry, isIndexed := indexedEntries[uniqueKey]
		isContentChange := false
		if !isDir {
			if !isIndexed || entryMtim.Equal(indexedEntry.modificationTime) {
				isContentChange = true
			}
		}

		readJobs <- syncJob{
			path:            filePath,
			isIndexed:       isIndexed,
			isContentChange: isContentChange,
			stat:            &entryStat,
			statT:           *entryStatT,
		}
	}

	return nil
}

// newDirWorker is route newDirJobs to the traverseNewDir function which starts the process of collecting the details for indexing the new directory
func newDirWorker(newDirJobs <-chan string, readJobs chan<- syncJob, wg *sync.WaitGroup, indexedEntries map[string]entryHeader) {
	defer wg.Done()
	for path := range newDirJobs {
		err := traverseNewDir(readJobs, path, indexedEntries)
		if err != nil {
			slog.Error("", "call", "syncer.traverseNewDir()", "err", err)
		}
	}
}

// traverseNewDir crawls a newly created directory and it's file system tree
// categorizes the entries and produces parameterized readjobs with the entries stat and stat_t details.
func traverseNewDir(
	readJobs chan<- syncJob,
	startPath string,
	indexedEntries map[string]entryHeader,
) error {
	err := filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				slog.Debug("", "call", "filepath.WalkDir() -> os.IsPermission()", "err", err)
			} else {
				slog.Error(fmt.Sprintf("failed to walk path %s", path), "call", "filepath.Walkdir()", "err", err)
			}
			return nil
		}

		if d.IsDir() && slices.Contains(config.Details.ExcludedEntries, filepath.Base(path)) {
			slog.Debug(fmt.Sprintf("excluded path %s", path), "call", "filepath.WalkDir() -> isDir && slices.Contains()")
			return filepath.SkipDir
		}

		entryStat, err := os.Lstat(path)
		if err != nil {
			slog.Error(fmt.Sprintf("failed on path %s", path), "call", "filepath.WalkDir() -> os.Lstat()", "err", err)
			return nil
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
				isContentChange = !entryMtim.Equal(indexedEntry.modificationTime)
			}
		}

		readJobs <- syncJob{
			path:            path,
			isIndexed:       isIndexed,
			isContentChange: isContentChange,
			stat:            &entryStat,
			statT:           *entryStatT,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("syncer.traverseNewDir() -> filepath.Walkdir() for startpath %s %w", startPath, err)
	}

	return nil
}

// traverseDirectories crawls the directory tree from top level and checks for mTim/cTim changes to identify if anything has changed within the directory
// if changes are identified, it produces a readJob to record the metadata of the direcotry itself and
// a scanJob to read the contents of the directory to identify what has changed at lower levels
// if the directory is not previously indexed (with the same unique key, i.e. dev_id+inode+path), it produces a newDirJob to perform a full scan of the directory tree below it
func traverseDirectories(
	scanJobs chan<- entryHeader,
	newDirJobs chan<- string,
	readJobs chan<- syncJob,
	startPath string,
	indexedEntries map[string]entryHeader,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	err := filepath.WalkDir(startPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if !os.IsPermission(err) {
				slog.Error(fmt.Sprintf("failed to walk path %s", path), "call", "filepath.Walkdir()", "err", err)
			}
			return nil
		}

		if d.IsDir() && slices.Contains(config.Details.ExcludedEntries, filepath.Base(path)) {
			return filepath.SkipDir
		}

		entryStat, err := os.Lstat(path)
		if err != nil {
			slog.Error(fmt.Sprintf("failed on path %s", path), "call", "filepath.WalkDir() -> os.Lstat()", "err", err)
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
				if !indexedEntry.modificationTime.Equal(mTim) || !indexedEntry.metaDataChangeTime.Equal(cTim) {
					scanJobs <- indexedEntry
					readJobs <- syncJob{
						path:            path,
						isIndexed:       true,
						isContentChange: false,
						stat:            &entryStat,
						statT:           *statT,
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		slog.Error("failed to walk directory", "call", "filepath.WalkDir()", "err", err)
	}
}
