package syncer

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"igloo/indexer"
)

// updateAfterSync takes the collected data from the sync processes and triggers db updates of the index
func updateAfterSync(syncDetails *syncCollection, con *sql.DB) {
	countOfDeletions := len(syncDetails.deletions)
	countOfNewEntries := len(syncDetails.newEntries)
	countOfUpdatesWContent := len(syncDetails.updatesWContent)
	countOfUpdatesWOContent := len(syncDetails.updatesWOContent)
	slog.Debug(fmt.Sprintf("Updating DB for: %d Deletions - %d New entries - %d Updates with content - %d Updates without content",
		countOfDeletions,
		countOfNewEntries,
		countOfUpdatesWContent,
		countOfUpdatesWOContent,
	))
	updateDBStart := time.Now()
	if countOfDeletions > 0 {
		deleteEntries(con, syncDetails.deletions)
	}
	if countOfNewEntries > 0 {
		indexer.WriteFullEntries(con, syncDetails.newEntries, nil)
	}
	if countOfUpdatesWContent > 0 {
		updateEntriesWithContent(con, syncDetails.updatesWContent)
	}
	if countOfUpdatesWOContent > 0 {
		updateEntriesWithoutContent(con, syncDetails.updatesWOContent)
	}
	elapsed := time.Since(updateDBStart)
	slog.Info(fmt.Sprintf("Updates to DB took: %s", elapsed), "call", "syncer.updateAfterSync()")
}

func updateEntriesWithContent(con *sql.DB, entryCollection []*indexer.EntryCollection) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("syncer.updateEntriesWithContent() -> con.Begin() %w", err)
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
		return fmt.Errorf("syncer.updateEntriesWithContent() -> transaction.Prepare() %w", err)
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
			return fmt.Errorf("syncer.pdateEntriesWithContent() -> statement.Exec() for path: %s %w", entry.FullPath, err)
		}
	}

	return transaction.Commit()
}

func updateEntriesWithoutContent(con *sql.DB, entryCollection []*indexer.EntryCollection) error {
	transaction, err := con.Begin()
	if err != nil {
		return fmt.Errorf("syncer.pdateEntriesWithoutContent() -> con.Begin() %w", err)
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
		return fmt.Errorf("syncer.updateEntriesWithoutContent() -> transaction.Prepare() %w", err)
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
			return fmt.Errorf("syncer.updateEntriesWithoutContent() -> statement.Exec() for path: %s %w", entry.FullPath, err)
		}
	}

	return transaction.Commit()
}
