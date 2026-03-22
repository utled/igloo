// Package maintain manages the continous syncing and updating/deletion of indexed file system entries
package maintain

import (
	"database/sql"
	"fmt"
	"igloo/config"
	"igloo/data"
	"igloo/db"
	"os"
	"path/filepath"
	"time"
)

func Start() error {
	config, err := config.GetConfig()
	if err != nil {
		fmt.Println(err)
	}
	scanCount := 10
	for scanCount > 0 {
		var startPath string
		var err error

		if scanCount%config.LargeSyncFrequenzy == 0 {
			startPath = config.LargeSyncPath
		} else {
			startPath = config.QuickSyncPath
		}
		homePath, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dbPath := filepath.Join(homePath, ".igloo", "igloo.db")

		con, err := db.CreateConnection(dbPath)
		if err != nil {
			return err
		}
		defer func(con *sql.DB) {
			err = db.CloseConnection(con)
			if err != nil {
				fmt.Println(err)
			}
		}(con)

		indexedEntries, err := data.GetIndexedEntries(con)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("Starting scan of: %s\n", startPath)
		startTime := time.Now()
		syncInfo := data.SyncInfo{}
		err = orchestrateScan(startPath, indexedEntries, &config, &syncInfo, con)
		if err != nil {
			return err
		}
		elapsed := time.Since(startTime)
		fmt.Printf("Scan completed in: %s\n", elapsed)
		
		countOfNewEntries := len(syncInfo.NewEntries)
		countOfUpdatesWContent := len(syncInfo.UpdatesWContent)
		countOfUpdatesWOContent := len(syncInfo.UpdatesWOContent)
		countOfDeletions := len(syncInfo.Deletions)
		fmt.Printf("Starting DB updates for:\n%d New entries\n%d Updates with content\n%d Updates without content\n%d Deletions\n",
			countOfNewEntries, 
			countOfUpdatesWContent, 
			countOfUpdatesWOContent, 
			countOfDeletions,
		)
		updateDBStart := time.Now()
		if countOfDeletions > 0 {
			data.DeleteEntries(con, syncInfo.Deletions)
		}
		if countOfNewEntries > 0 {
			data.WriteFullEntries(con, syncInfo.NewEntries)
			for _, entry := range syncInfo.NewEntries {
				fmt.Println("new entry:", entry.FullPath)
			}
		}
		if countOfUpdatesWContent > 0 {
			data.UpdateEntriesWithContent(con, syncInfo.UpdatesWContent)
			for _, entry := range syncInfo.UpdatesWContent {
				fmt.Println("entry with content:", entry.FullPath)
			}
		}
		if countOfUpdatesWOContent > 0 {
			data.UpdateEntriesWithoutContent(con, syncInfo.UpdatesWOContent)
			for _, entry := range syncInfo.UpdatesWOContent {
				fmt.Println("entry without content:", entry.FullPath)
			}
		}
		elapsed = time.Since(updateDBStart)
		fmt.Printf("Updates to DB took: %s\n", elapsed)
		
		time.Sleep(1 * time.Second)

		if scanCount == 1 {
			scanCount = 10
		} else {
			scanCount--
		}

	}

	return nil
}
