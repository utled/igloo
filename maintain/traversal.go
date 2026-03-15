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
	"sync"
	"syscall"
	"time"
)

func traverseNewDir(readJobs chan<- data.SyncJob, startPath string, config *data.Config, con *sql.DB) error {
	fmt.Println("Traversing new dir: ", startPath)
	inodeMappedEntries, err := data.GetInodeMappedEntries(con)
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
		if inode, ok := inodeMappedEntries[entryStatT.Ino]; ok {
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

	return nil
}

func traverseDirectories(
	scanJobs chan<- data.InodeHeader,
	newDirJobs chan<- string,
	readJobs chan<- data.SyncJob,
	startPath string,
	inodeMappedEntries map[uint64]data.InodeHeader,
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

		if d.IsDir() {
			if _, ok := inodeMappedEntries[statT.Ino]; !ok {
				newDirJobs <- path
			} else {
				for inode, values := range inodeMappedEntries {
					if inode != statT.Ino {
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
