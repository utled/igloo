package indexer

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

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

// WriteFullEntries unwraps a slice of pointers to EntryCollection instances and produces a bulk sql write transaction before commiting the write to DB
// the *sync.WaitGroup parameter is included to be able to use the function within a goroutine, but can be set to nil if the write is used synchronous
func WriteFullEntries(con *sql.DB, entryCollection []*EntryCollection, wg *sync.WaitGroup) error {
	if wg != nil {
		defer wg.Done()
	}
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("data.WriteFullEntries() -> con.Begin() %w", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare(`insert into entries(
		dev_id,
		inode,
    path,
		parent_directory,
		name,
		is_dir,
		size,
		modification_time,
		access_time,
		metadata_change_time,
		owner_id,
		group_id,
		extension,
		content_snippet,
		full_text,
    line_count_total,
    line_count_w_content)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("data.WriteFullEntries() -> transcation.Prepare() %w", err)
	}
	defer statement.Close()

	for _, entry := range entryCollection {
		_, err := statement.Exec(
			entry.DevID,
			entry.Inode,
			entry.FullPath,
			entry.ParentDirID,
			entry.Name,
			entry.IsDir,
			entry.Size,
			entry.ModificationTime,
			entry.AccessTime,
			entry.MetaDataChangeTime,
			entry.OwnerID,
			entry.GroupID,
			entry.Extension,
			entry.ContentSnippet,
			entry.FullTextIndex,
			entry.LineCountTotal,
			entry.LineCountWithContent,
		)
		if err != nil {
			return fmt.Errorf("data.WriteFullEntries() -> statement.Exec() for path: %s %w", entry.FullPath, err)
		}
	}

	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("data.WriteFullEntries() -> transaction.Commit() %w", err)
	}

	return nil
}
