// Package data holds the definitions of all data structures used in the program
// as well ass provides writers and getters to interact with external data sources
package data

import (
	"os"
	"sync"
	"syscall"
	"time"
)

type CollectedInfo struct {
	EntryDetails []*EntryCollection
	Mu           sync.Mutex
}

type SyncInfo struct {
	Deletions        []*DeletionJob
	NewEntries       []*EntryCollection
	UpdatesWContent  []*EntryCollection
	UpdatesWOContent []*EntryCollection
	Mu               sync.Mutex
}

type EntryCollection struct {
	DevID       uint64
	Inode       uint64
	FullPath    string
	ParentDirID string
	Name        string
	IsDir       bool
	Size        int64
	// creationTime       int64 // Btim (not included syscall.Stat_t)
	ModificationTime     time.Time // os.fileStat.sys.Mtim.Sec + Mtim.Nsec
	AccessTime           time.Time // os.fileStat.sys.Atim.Sec + Atim.Nsec
	MetaDataChangeTime   time.Time // os.fileStat.sys.Ctim.Sec + Ctim.Nsec
	OwnerID              uint32    // os.fileStat.sys.Uid
	GroupID              uint32    // os.fileStat.sys.Gid
	Extension            string
	ContentSnippet       []byte // short extract of the files content. <= [:500]
	FullTextIndex        []byte // the complete textual content of a document, stored in separate Full-Text Search index
	LineCountTotal       int
	LineCountWithContent int
}

type ReadJob struct {
	Path string
	Stat *os.FileInfo
}

type SyncJob struct {
	Path            string
	IsIndexed       bool
	IsContentChange bool
	Stat            *os.FileInfo
	StatT           syscall.Stat_t
}

type DeletionJob struct {
	UniqueKey string
	DevID     uint64
	Inode     uint64
	Path      string
}

type EntryHeader struct {
	DevID              uint64
	Inode              uint64
	Path               string
	ModificationTime   time.Time
	MetaDataChangeTime time.Time
}
