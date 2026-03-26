package initial

import (
	"database/sql"
	"fmt"
	"igloo/data"
	"igloo/db"
	"log/slog"
)

func writeFullIndex(theWorks *data.CollectedInfo) error {
	con, err := db.CreateConnection()
	if err != nil {
		return fmt.Errorf("initial.writeFullIndex() -> db.CreateConnection() %w", err)
	}
	defer func(con *sql.DB) {
		err = db.CloseConnection(con)
		if err != nil {
			slog.Error("failed to close db connection", "call", "db.CloseConnection()", "err", err)
		}
	}(con)

	err = data.ClearExistingData(con)
	if err != nil {
		return fmt.Errorf("initial.writeFullIndex() -> data.ClearExistingData() %w", err)
	}

	err = data.WriteFullEntries(con, theWorks.EntryDetails)
	if err != nil {
		return fmt.Errorf("initial.writeFullIndex() -> data.WriteFullEntries() %w", err)
	}

	return nil
}
