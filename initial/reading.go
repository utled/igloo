package initial

import (
	"bytes"
	"fmt"
	"igloo/data"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"syscall"
	"time"
)

func readDir(path string, stat *os.FileInfo, theWorks *data.CollectedInfo, isRoot bool) {
	entry := data.EntryCollection{}

	dirStat := *stat

	entry.FullPath = path
	if !isRoot {
		entry.ParentDirID = filepath.Dir(path)
	}
	entry.Name = filepath.Base(path)
	entry.IsDir = true
	entry.Size = dirStat.Size()

	statT := dirStat.Sys().(*syscall.Stat_t)
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
	theWorks.NumOfDirectories += 1
	theWorks.EntryDetails = append(theWorks.EntryDetails, &entry)
	theWorks.Mu.Unlock()
}

func readFile(filename string, stat *os.FileInfo, theWorks *data.CollectedInfo, config *data.Config) {
	entry := data.EntryCollection{}
	contentsRead := false
	fileStat := *stat

	if fileStat.Mode().Type()&os.ModeSymlink == 0 && slices.Contains(config.ContentFileTypes, filepath.Ext(filename)) {
		contents, err := os.ReadFile(filename)
		if err != nil {
			fmt.Println(err)
		}
		contentsRead = true
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


	entry.FullPath = filename
	entry.ParentDirID = filepath.Dir(filename)
	entry.Name = filepath.Base(filename)
	entry.IsDir = false
	entry.Size = fileStat.Size()

	statT := fileStat.Sys().(*syscall.Stat_t)
	entry.Inode = statT.Ino
	entry.ModificationTime = time.Unix(statT.Mtim.Sec, statT.Mtim.Nsec)
	entry.AccessTime = time.Unix(statT.Atim.Sec, statT.Atim.Nsec)
	entry.MetaDataChangeTime = time.Unix(statT.Ctim.Sec, statT.Ctim.Nsec)

	entry.OwnerID = statT.Uid
	entry.GroupID = statT.Gid

	theWorks.Mu.Lock()
	theWorks.NumOfFiles += 1
	if contentsRead {
		theWorks.NumOfFilesWithContent += 1
	}
	theWorks.EntryDetails = append(theWorks.EntryDetails, &entry)
	theWorks.Mu.Unlock()
}
