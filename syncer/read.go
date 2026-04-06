package syncer

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"syscall"
	"time"

	"igloo/config"
	"igloo/indexer"
)

type syncJob struct {
	path            string
	isIndexed       bool
	isContentChange bool
	stat            *os.FileInfo
	statT           syscall.Stat_t
}

func readWorker(readJobs <-chan syncJob, syncDetails *syncCollection, wg *sync.WaitGroup) {
	defer wg.Done()
	counter := 0
	for job := range readJobs {
		readEntry(job, syncDetails)
		counter++
		if counter == 10 {
			time.Sleep(1 * time.Millisecond)
			counter = 0
		}
	}
}

// readEntry performs the actual collection of a file system entries' metadata
// and reads file contents for filetypes defined in the program config
// based on the readjobs' parameteres it decides what output struct to store the data in (new entry, update with content, update without content)
func readEntry(syncJob syncJob, syncDetails *syncCollection) {
	entryStat := *syncJob.stat
	statT := syncJob.statT

	entry := indexer.EntryCollection{}

	entry.FullPath = syncJob.path
	entry.ParentDirID = filepath.Dir(syncJob.path)
	entry.Name = filepath.Base(syncJob.path)
	entry.IsDir = entryStat.IsDir()
	entry.Size = entryStat.Size()

	entry.DevID = statT.Dev
	entry.Inode = statT.Ino
	entry.ModificationTime = time.Unix(statT.Mtim.Sec, statT.Mtim.Nsec)
	entry.AccessTime = time.Unix(statT.Atim.Sec, statT.Atim.Nsec)
	entry.MetaDataChangeTime = time.Unix(statT.Ctim.Sec, statT.Ctim.Nsec)

	entry.OwnerID = statT.Uid
	entry.GroupID = statT.Gid

	if !entryStat.IsDir() && entryStat.Mode().Type()&os.ModeSymlink == 0 {
		if slices.Contains(config.Details.ContentFileTypes, filepath.Ext(syncJob.path)) && syncJob.isContentChange {
			contents, err := os.ReadFile(syncJob.path)
			if err != nil {
				slog.Error(fmt.Sprintf("failed to read file %s", syncJob.path), "call", "os.ReadFile()", "err", err)
			}
			lineCountTotal := bytes.Count(contents, []byte("\n"))
			blankLines := bytes.Count(contents, []byte("\n\n"))
			lineCountWithContent := lineCountTotal - blankLines

			if len(contents) < 500 {
				entry.ContentSnippet = contents
			} else {
				entry.ContentSnippet = contents[:500]
			}

			contents = bytes.ReplaceAll(contents, []byte("\n"), []byte(" "))
			contents = bytes.ReplaceAll(contents, []byte("\r"), []byte(" "))
			contents = bytes.ReplaceAll(contents, []byte("\t"), []byte(" "))

			regExCleanup := regexp.MustCompile(`[\p{C}\p{Zl}\p{Zp}]`)
			contents = regExCleanup.ReplaceAll(contents, []byte(" "))
			contents = regexp.MustCompile(`\s+`).ReplaceAll(contents, []byte(" "))

			entry.FullTextIndex = contents
			entry.LineCountTotal = lineCountTotal
			entry.LineCountWithContent = lineCountWithContent
		}
	}
	if !syncJob.isIndexed {
		syncDetails.mu.Lock()
		defer syncDetails.mu.Unlock()
		syncDetails.newEntries = append(syncDetails.newEntries, &entry)
		return
	}
	if syncJob.isContentChange {
		syncDetails.mu.Lock()
		defer syncDetails.mu.Unlock()
		syncDetails.updatesWContent = append(syncDetails.updatesWContent, &entry)
		return
	}
	syncDetails.mu.Lock()
	defer syncDetails.mu.Unlock()
	syncDetails.updatesWOContent = append(syncDetails.updatesWOContent, &entry)
}
