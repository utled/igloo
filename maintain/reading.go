package maintain

import (
	"bytes"
	"fmt"
	"igloo/data"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"
)

// readEntry performs the actual collection of a file system entries' metadata
// and reads file contents for filetypes defined in the program config
// based on the readjobs' parameteres it decides what output struct to store the data in (new entry, update with content, update without content)
func readEntry(syncJob data.SyncJob, config *data.Config, syncInfo *data.SyncInfo) {
	entryStat := *syncJob.Stat
	statT := syncJob.StatT

	entry := data.EntryCollection{}

	entry.FullPath = syncJob.Path
	entry.ParentDirID = filepath.Dir(syncJob.Path)
	entry.Name = filepath.Base(syncJob.Path)
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
		if slices.Contains(config.ContentFileTypes, filepath.Ext(syncJob.Path)) && syncJob.IsContentChange {
			contents, err := os.ReadFile(syncJob.Path)
			if err != nil {
				fmt.Println(err)
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
	if !syncJob.IsIndexed {
		syncInfo.Mu.Lock()
		defer syncInfo.Mu.Unlock()
		syncInfo.NewEntries = append(syncInfo.NewEntries, &entry)
		return
	}
	if syncJob.IsContentChange {
		syncInfo.Mu.Lock()
		defer syncInfo.Mu.Unlock()
		syncInfo.UpdatesWContent = append(syncInfo.UpdatesWContent, &entry)
		return
	}
	syncInfo.Mu.Lock()
	defer syncInfo.Mu.Unlock()
	syncInfo.UpdatesWOContent = append(syncInfo.UpdatesWOContent, &entry)
}
