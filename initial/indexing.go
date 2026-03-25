package initial

import (
	"database/sql"
	"fmt"
	"igloo/data"
	"igloo/db"
)

func writeFullIndex(theWorks *data.CollectedInfo) error {
	con, err := db.CreateConnection()
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

	return nil
}
