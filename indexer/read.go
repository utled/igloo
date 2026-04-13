package indexer

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"syscall"
	"time"

	"igloo/config"
)

type readJob struct {
	path string
	stat *os.FileInfo
}

func readWorker(readJobs <-chan readJob, batchJobs chan<- *EntryCollection, wg *sync.WaitGroup) {
	defer wg.Done()
	counter := 0
	for job := range readJobs {
		readEntry(batchJobs, job.path, job.stat, false)
		counter++
		if counter == 10 {
			time.Sleep(2 * time.Millisecond)
			counter = 0
		}
	}
}

// readEntry reads the entry details from Stat, collects and reads Stat_t
// and reads the file contents for the filetypes defined in the config file
// with the details collected it sends the data through the batchJobs channel to be batched before writing to DB.
func readEntry(batchJobs chan<- *EntryCollection, path string, stat *os.FileInfo, isRoot bool) {
	entry := EntryCollection{}
	entryStat := *stat
	entryStatT := entryStat.Sys().(*syscall.Stat_t)

	if !entryStat.IsDir() {
		if entryStat.Mode().Type()&os.ModeSymlink == 0 && slices.Contains(config.Details.ContentFileTypes, filepath.Ext(path)) {
			contents, err := os.ReadFile(path)
			if err != nil {
				if !os.IsPermission(err) {
					slog.Error("failed to read file", "call", "os.ReadFile()", "err", err)
				}
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
	entry.Name = entryStat.Name()
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

	batchJobs <- &entry
}
