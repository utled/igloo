package maintain

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"igloo/data"
	"syscall"
	"time"
)

func readEntry(syncJob data.SyncJob, config *data.Config, con *sql.DB) {
	entryStat, err := os.Stat(syncJob.Path)
	if err != nil {
		fmt.Println(err)
	}

	entry := data.EntryCollection{}

	entry.FullPath = syncJob.Path
	entry.ParentDirID = filepath.Dir(syncJob.Path)
	entry.Name = filepath.Base(syncJob.Path)
	entry.IsDir = entryStat.IsDir()
	entry.Size = entryStat.Size()

	statT := entryStat.Sys().(*syscall.Stat_t)

	entry.DevID = statT.Dev
	entry.Inode = statT.Ino
	entry.ModificationTime = time.Unix(statT.Mtim.Sec, statT.Mtim.Nsec)
	entry.AccessTime = time.Unix(statT.Atim.Sec, statT.Atim.Nsec)
	entry.MetaDataChangeTime = time.Unix(statT.Ctim.Sec, statT.Ctim.Nsec)

	entry.OwnerID = statT.Uid
	entry.GroupID = statT.Gid

	if !entryStat.IsDir() {
		if slices.Contains(config.ContentFileTypes, filepath.Ext(syncJob.Path)) && syncJob.IsContentChange {
			contents, err := os.ReadFile(syncJob.Path)
			if err != nil {
				log.Fatal(err)
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

	entryCollection := make([]*data.EntryCollection, 1)
	entryCollection[0] = &entry
	if !syncJob.IsIndexed {
		err := data.WriteFullEntries(con, entryCollection)
		if err != nil {
			fmt.Println("error writing: ", entry.FullPath, err)
		}
		return
	}
	if syncJob.IsContentChange {
		err := data.UpdateEntriesWithContent(con, entryCollection)
		if err != nil {
			fmt.Println(err)
		}
		return
	}
	err = data.UpdateEntriesWithoutContent(con, entryCollection)
	if err != nil {
		fmt.Println(err)
	}
}
