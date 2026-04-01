package initial

import (
	"bytes"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"syscall"
	"time"

	"igloo/data"
	"igloo/utils"
)

func batchWorker(batchJobs <-chan *data.EntryCollection, countChan chan<- int, batchSize int, wg *sync.WaitGroup, con *sql.DB) {
	defer wg.Done()

	var batchedEntries []*data.EntryCollection
	var batchCount int
	for job := range batchJobs {
		batchedEntries = append(batchedEntries, job)
		batchCount++
		if batchCount == batchSize {
			err := data.WriteFullEntries(con, batchedEntries)
			if err != nil {
				slog.Error("on looped batch", "call", "data.WriteFullEntries()", "err", err)
			}
			countChan <- batchCount
			batchedEntries = nil
			batchCount = 0
		}
	}
	err := data.WriteFullEntries(con, batchedEntries)
	if err != nil {
		slog.Error("on final batch", "call", "data.WriteFullEntries()", "err", err)
	}
	countChan <- batchCount
}

func readWorker(readJobs <-chan data.ReadJob, batchJobs chan<- *data.EntryCollection, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range readJobs {
		readEntry(batchJobs, job.Path, job.Stat, false)
	}
}

func readEntry(batchJobs chan<- *data.EntryCollection, path string, stat *os.FileInfo, isRoot bool) {
	entry := data.EntryCollection{}
	entryStat := *stat
	entryStatT := entryStat.Sys().(*syscall.Stat_t)

	if !entryStat.IsDir() {
		if entryStat.Mode().Type()&os.ModeSymlink == 0 && slices.Contains(utils.Config.ContentFileTypes, filepath.Ext(path)) {
			contents, err := os.ReadFile(path)
			if err != nil {
				slog.Error("failed to read file", "call", "os.ReadFile()", "err", err)
				return
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

	entry.FullPath = path
	if !isRoot {
		entry.ParentDirID = filepath.Dir(path)
	}
	entry.Name = filepath.Base(path)
	entry.IsDir = entryStat.IsDir()
	entry.Size = entryStat.Size()

	entry.DevID = entryStatT.Dev
	entry.Inode = entryStatT.Ino
	entry.ModificationTime = time.Unix(entryStatT.Mtim.Sec, entryStatT.Mtim.Nsec)
	entry.AccessTime = time.Unix(entryStatT.Atim.Sec, entryStatT.Atim.Nsec)
	entry.MetaDataChangeTime = time.Unix(entryStatT.Ctim.Sec, entryStatT.Ctim.Nsec)

	entry.OwnerID = entryStatT.Uid
	entry.GroupID = entryStatT.Gid
	entry.Extension = filepath.Ext(entry.Name)
	entry.FileType = filepath.Ext(entry.Name)

	batchJobs <- &entry
}
