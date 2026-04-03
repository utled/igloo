package indexer

import (
	"database/sql"
	"fmt"
)

// checkTableExists checks that the tables that is to be cleared actually exist in the db
func checkTableExists(con *sql.DB, tableName string) (tableExists bool, err error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name=?`

	row := con.QueryRow(query, tableName)

	var name string
	err = row.Scan(&name)

	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("checkTableExists() -> row.Scan() %w", err)
	default:
		return true, nil
	}
}

// clearExistingData calls the checkTableExists function and deletes all entries from table
// it is to be used before starting a full index refresh
func clearExistingData(con *sql.DB) error {
	entriesExist, err := checkTableExists(con, "entries")
	if err != nil {
		return fmt.Errorf("ClearExistingData() -> checkTableExists() %w", err)
	} else if entriesExist {
		query := `delete from entries;`
		_, err = con.Exec(query)
		if err != nil {
			return fmt.Errorf("ClearExistingData() -> con.Exex() %w", err)
		}
	}
	return nil
}
