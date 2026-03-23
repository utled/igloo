package data

import (
	"os"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	LargeSyncPath      string   `json:"LargeSyncPath"`      // defaults to system root directory
	QuickSyncPath      string   `json:"QuickSyncPath"`      // defaults to /home/user/
	LargeSyncFrequenzy int      `json:"LargeSyncFrequenzy"` // what nth sync loop runs the full sync scan
	WaitBetweenSyncs   int      `json:"WaitBetweenSyncs"`   // defaults to 1 second
	ExcludedEntries    []string `json:"ExcludedEntries"`    // what files and directories are excluded from being indexed
	ContentFileTypes   []string `json:"ContentFileTypes"`   // what file types does the index capture the contents for to allow content based searches of the index
}

type CollectedInfo struct {
	ScanStart             time.Time
	ScanEnd               time.Time
	ScanDuration          time.Duration
	IndexingCompleted     bool
	NumOfFiles            int
	NumOfDirectories      int
	NumOfFilesWithContent int
	NumOfIgnoredEntries   int
	EntryDetails          []*EntryCollection
	NotRegistered         []*NotAccessedPaths
	Mu                    sync.Mutex
}

type SyncInfo struct {
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
	FileType             string // MIME type
	ContentSnippet       []byte // short extract of the files content. <= [:500]
	FullTextIndex        []byte // the complete textual content of a document, stored in separate Full-Text Search index
	LineCountTotal       int
	LineCountWithContent int
}

type NotAccessedPaths struct {
	Path string
	Err  string
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

type DeletionInfo struct {
	Deletions        []*DeletionJob
	Mu               sync.Mutex
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
