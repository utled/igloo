package data

import (
	"database/sql"
	"fmt"
)

func checkTableExists(con *sql.DB, tableName string) (bool, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`

	row := con.QueryRow(query, tableName)

	var name string
	err := row.Scan(&name)

	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("checkTableExists error: %w", err)
	default:
		return true, nil
	}
}

func ClearExistingData(con *sql.DB) error {
	entriesExist, err := checkTableExists(con, "entries")
	if err != nil {
		return err
	} else if entriesExist {
		query := `delete from entries;`
		_, err = con.Exec(query)
		if err != nil {
			return fmt.Errorf("could not clear existing data with query: %s\n%w", query, err)
		}
	}

	ignoredEntriesExist, err := checkTableExists(con, "entries")
	if err != nil {
		return err
	} else if ignoredEntriesExist {
		query := `delete from ignored_entries;`
		_, err = con.Exec(query)
		if err != nil {
			return fmt.Errorf("could not clear existing data with query: %s\n%w", query, err)
		}
	}

	return nil
}

func WriteFullEntries(con *sql.DB, entryCollection []*EntryCollection) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for db write:%v", err)
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
					filetype,
					content_snippet,
					full_text,
                    line_count_total,
                    line_count_w_content)
					values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare execution statement for db write:%v", err)
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
			entry.FileType,
			entry.ContentSnippet,
			entry.FullTextIndex,
			entry.LineCountTotal,
			entry.LineCountWithContent,
		)
		if err != nil {
			return fmt.Errorf("could not add entry %s to db write statement: \n%w", entry.FullPath, err)
		}
	}

	return transaction.Commit()
}

func UpdateEntriesWithContent(con *sql.DB, entryCollection []*EntryCollection) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for db write:%v", err)
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
		filetype = ?,
		content_snippet = ?,
		full_text = ?,
    line_count_total = ?,
    line_count_w_content = ?
		where dev_id = ? and inode = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare execution statement for db write:%v", err)
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
			entry.FileType,
			entry.ContentSnippet,
			entry.FullTextIndex,
			entry.LineCountTotal,
			entry.LineCountWithContent,
			entry.DevID,
			entry.Inode,
		)
		if err != nil {
			return fmt.Errorf("(with content) could not add entry %s to db update statement: \n%w", entry.FullPath, err)
		}
	}

	return transaction.Commit()
}

func UpdateEntriesWithoutContent(con *sql.DB, entryCollection []*EntryCollection) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for db write:%v", err)
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
		filetype = ?
		where dev_id = ? and inode = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare execution statement for db write:%v", err)
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
			entry.FileType,
			entry.DevID,
			entry.Inode,
		)
		if err != nil {
			return fmt.Errorf("(without content) could not add entry %s to db update statement: \n%w", entry.FullPath, err)
		}
	}

	return transaction.Commit()
}

func WriteNotRegisteredEntries(con *sql.DB, notRegistered []*NotAccessedPaths) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for not registered entries:%v", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare(`insert into ignored_entries(path, error) values(?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement for not registered entries:%v", err)
	}
	defer statement.Close()

	for _, entry := range notRegistered {
		_, err := statement.Exec(entry.Path, entry.Err)
		if err != nil {
			return fmt.Errorf("could not add not registered entries to update statement: %s\n%w", err)
		}
	}

	return transaction.Commit()
}

func WriteScanRecord(con *sql.DB, theWorks *CollectedInfo) error {
	query := `insert into full_scans(
                    scan_start,
					scan_end,
					scan_duration,
					directory_count,
				    file_count,
				    file_w_content_count,
				    ignored_entries_count,
				    indexing_completed)
					values (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := con.Exec(
		query,
		theWorks.ScanStart,
		theWorks.ScanEnd,
		theWorks.ScanDuration,
		theWorks.NumOfDirectories,
		theWorks.NumOfFiles,
		theWorks.NumOfFilesWithContent,
		theWorks.NumOfIgnoredEntries,
		theWorks.IndexingCompleted)
	if err != nil {
		return fmt.Errorf("could not write entry to database: %s\n%w", query, err)
	}

	return nil
}

func DeleteEntries(con *sql.DB, deletionEntries []*DeletionJob) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for deletions:%v", err)
	}
	defer transaction.Rollback()

	statement, err := transaction.Prepare(`delete from entries where dev_id = ? and inode = ? and path = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare execution statement for deletions:%v", err)
	}
	defer statement.Close()

	for _, entry := range deletionEntries {
		_, err := statement.Exec(
			entry.DevID,
			entry.Inode,
			entry.Path,
		)
		if err != nil {
			return fmt.Errorf("could not add entry %s to deletion statement: \n%w", entry.UniqueKey, err)
		}
	}

	return transaction.Commit()
}
