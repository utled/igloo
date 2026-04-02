package data

import (
	"database/sql"
	"fmt"
	"sync"
)

func checkTableExists(con *sql.DB, tableName string) (tableExists bool, err error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`

	row := con.QueryRow(query, tableName)

	var name string
	err = row.Scan(&name)

	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("data.checkTableExists() -> row.Scan() %w", err)
	default:
		return true, nil
	}
}

func ClearExistingData(con *sql.DB) error {
	entriesExist, err := checkTableExists(con, "entries")
	if err != nil {
		return fmt.Errorf("data.ClearExistingData() -> checkTableExists() %w", err)
	} else if entriesExist {
		query := `delete from entries;`
		_, err = con.Exec(query)
		if err != nil {
			return fmt.Errorf("data.ClearExistingData() -> con.Exex() %w", err)
		}
	}
	return nil
}

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

func UpdateEntriesWithContent(con *sql.DB, entryCollection []*EntryCollection) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("data.UpdateEntriesWithContent() -> con.Begin() %w", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare(`update entries 
		set 
		path = ?,
		parent_directory = ?,
		name = ?,
		is_dir = ?,
		size = ?,
		modification_time = ?,
		access_time = ?,
		metadata_change_time = ?,
		owner_id = ?,
		group_id = ?,
		extension = ?,
		content_snippet = ?,
		full_text = ?,
    line_count_total = ?,
    line_count_w_content = ?
		where dev_id = ? and inode = ?`)
	if err != nil {
		return fmt.Errorf("data.UpdateEntriesWithContent() -> transaction.Prepare() %w", err)
	}
	defer statement.Close()

	for _, entry := range entryCollection {
		_, err := statement.Exec(
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
			entry.DevID,
			entry.Inode,
		)
		if err != nil {
			return fmt.Errorf("data.UpdateEntriesWithContent() -> statement.Exec() for path: %s %w", entry.FullPath, err)
		}
	}

	return transaction.Commit()
}

func UpdateEntriesWithoutContent(con *sql.DB, entryCollection []*EntryCollection) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("data.UpdateEntriesWithoutContent() -> con.Begin() %w", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare(`update entries set 
		path = ?,
		parent_directory = ?,
		name = ?,
		is_dir = ?,
		size = ?,
		modification_time = ?,
		access_time = ?,
		metadata_change_time = ?,
		owner_id = ?,
		group_id = ?,
		extension = ?,
		where dev_id = ? and inode = ?`)
	if err != nil {
		return fmt.Errorf("data.UpdateEntriesWithoutContent() -> transaction.Prepare() %w", err)
	}
	defer statement.Close()

	for _, entry := range entryCollection {
		_, err := statement.Exec(
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
			entry.DevID,
			entry.Inode,
		)
		if err != nil {
			return fmt.Errorf("data.UpdateEntriesWithoutContent() -> statement.Exec() for path: %s %w", entry.FullPath, err)
		}
	}

	return transaction.Commit()
}

func DeleteEntries(con *sql.DB, deletionEntries []*DeletionJob) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("data.DeleteEntries() -> con.Begin() %w", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare(`delete from entries where dev_id = ? and inode = ? and path = ?`)
	if err != nil {
		return fmt.Errorf("data.DeleteEntries() -> transaction.Prepare() %w", err)
	}
	defer statement.Close()

	for _, entry := range deletionEntries {
		_, err := statement.Exec(
			entry.DevID,
			entry.Inode,
			entry.Path,
		)
		if err != nil {
			return fmt.Errorf("data.DeleteEntries() -> statement.Exec() for entry key: %s %w", entry.UniqueKey, err)
		}
	}

	return transaction.Commit()
}
