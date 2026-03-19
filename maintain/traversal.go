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
	fmt.Println("Traversing new dir: ", startPath)
	uniqueMappedEntries, err := data.GetUniqueMappedEntries(con)
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

		var syncJob data.SyncJob
		entryStatT := entryStat.Sys().(*syscall.Stat_t)
		uniqueKey := strconv.Itoa(int(entryStatT.Dev)) + strconv.Itoa(int(entryStatT.Ino)) + path
		if inode, ok := uniqueMappedEntries[uniqueKey]; ok {
			entryMtim := time.Unix(entryStatT.Mtim.Sec, entryStatT.Mtim.Nsec)
			indexedMtim := inode.ModificationTime
			if entryStat.IsDir() || entryMtim.Equal(indexedMtim) {
				syncJob = data.SyncJob{Path: path, IsIndexed: true, IsContentChange: false}
			} else {
				syncJob = data.SyncJob{Path: path, IsIndexed: true, IsContentChange: true}
			}
		} else {
			if entryStat.IsDir() {
				syncJob = data.SyncJob{Path: path, IsIndexed: false, IsContentChange: false}
			} else {
				syncJob = data.SyncJob{Path: path, IsIndexed: false, IsContentChange: true}
			}
		}
		readJobs <- syncJob
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to traverse new directory:%v", err)
	}

	return nil
}

func traverseDirectories(
	scanJobs chan<- data.EntryHeader,
	newDirJobs chan<- string,
	readJobs chan<- data.SyncJob,
	startPath string,
	uniqueMappedEntries map[string]data.EntryHeader,
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
			if _, ok := uniqueMappedEntries[uniqueKey]; !ok {
				newDirJobs <- path
			} else {
				for key, values := range uniqueMappedEntries {
					if key != uniqueKey {
						continue
					}
					mTim := time.Unix(statT.Mtim.Sec, statT.Mtim.Nsec)
					cTim := time.Unix(statT.Ctim.Sec, statT.Ctim.Nsec)
					if !values.ModificationTime.Equal(mTim) || !values.MetaDataChangeTime.Equal(cTim) {
						readJobs <- data.SyncJob{Path: path, IsIndexed: true, IsContentChange: false}
						scanJobs <- values
						continue
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
