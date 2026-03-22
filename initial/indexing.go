package initial

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"igloo/data"
	"igloo/db"
)

func updateFullIndex(theWorks *data.CollectedInfo) error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
	}
	dbPath := filepath.Join(homePath, ".igloo", "igloo.db")
	con, err := db.CreateConnection(dbPath)
	if err != nil {
		fmt.Println(err)
	}
	defer func(con *sql.DB) {
		err = db.CloseConnection(con)
		if err != nil {
			fmt.Println(err)
		}
	}(con)

	err = data.ClearExistingData(con)
	if err != nil {
		return err
	}

	err = data.WriteFullEntries(con, theWorks.EntryDetails)
	if err != nil {
		return err
	}

	err = data.WriteNotRegisteredEntries(con, theWorks.NotRegistered)
	if err != nil {
		return err
	}

	theWorks.IndexingCompleted = true

	err = data.WriteScanRecord(con, theWorks)
	if err != nil {
		return err
	}

	return nil
}
