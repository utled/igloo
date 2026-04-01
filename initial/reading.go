package initial

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

	"igloo/data"
	"igloo/utils"
)

func readWorker(readJobs <-chan data.ReadJob, wg *sync.WaitGroup, theWorks *data.CollectedInfo) {
	defer wg.Done()
	for job := range readJobs {
		readEntry(job.Path, job.Stat, theWorks)
	}
}

func readEntry(filename string, stat *os.FileInfo, theWorks *data.CollectedInfo) {
	entry := data.EntryCollection{}
	fileStat := *stat

	if !fileStat.IsDir() {
		if fileStat.Mode().Type()&os.ModeSymlink == 0 && slices.Contains(utils.Config.ContentFileTypes, filepath.Ext(filename)) {
			contents, err := os.ReadFile(filename)
			if err != nil {
				slog.Error("failed to read file", "call", "os.ReadFile()", "err", err)
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

	entry.FullPath = filename
	entry.ParentDirID = filepath.Dir(filename)
	entry.Name = filepath.Base(filename)
	entry.IsDir = false
	entry.Size = fileStat.Size()

	statT := fileStat.Sys().(*syscall.Stat_t)
	entry.DevID = statT.Dev
	entry.Inode = statT.Ino
	entry.ModificationTime = time.Unix(statT.Mtim.Sec, statT.Mtim.Nsec)
	entry.AccessTime = time.Unix(statT.Atim.Sec, statT.Atim.Nsec)
	entry.MetaDataChangeTime = time.Unix(statT.Ctim.Sec, statT.Ctim.Nsec)

	entry.OwnerID = statT.Uid
	entry.GroupID = statT.Gid
	entry.Extension = filepath.Ext(entry.Name)
	entry.FileType = filepath.Ext(entry.Name)

	theWorks.Mu.Lock()
	theWorks.EntryDetails = append(theWorks.EntryDetails, &entry)
	theWorks.Mu.Unlock()
}
