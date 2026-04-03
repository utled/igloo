// Package db provides connection and initialization of external database
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func InitializeDB(servicePath string) error {
	db, err := CreateConnection()
	if err != nil {
		return fmt.Errorf("db.InitializwDB() %w", err)
	}
	defer CloseConnection(db)

	err = createTable(db)
	if err != nil {
		return fmt.Errorf("db.InitializeDB() -> createTable() %w", err)
	}

	return nil
}

func createTable(db *sql.DB) error {
	tableStatement := `create table if not exists entries (
				dev_id int not null,
    		inode int not null,
    		path text not null,
    		parent_directory text,
    		name text,
    		is_dir boolean,
    		size int,
    		modification_time datetime,
    		access_time datetime,
    		metadata_change_time datetime,
    		owner_id int,
    		group_id int,
    		extension text,
    		content_snippet text,
    		full_text text,
    		line_count_total int,
    		line_count_w_content int,
				primary key(dev_id, inode, path)
		) without rowid;`

	_, err := db.Exec(tableStatement)
	if err != nil {
		return fmt.Errorf("db.createTable() for statement: %s %w", tableStatement, err)
	}

	return nil
}

func CreateConnection() (*sql.DB, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(homePath, ".igloo", "igloo.db")
	dbPathWithWAL := dbPath+"?_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dbPathWithWAL)
	if err != nil {
		return nil, fmt.Errorf("db.CreateConnection() -> sql.Open() for path %s %w", dbPathWithWAL, err)
	}

	return db, nil
}

func CloseConnection(db *sql.DB) error {
	err := db.Close()
	if err != nil {
		return fmt.Errorf("db.CloseConnection() %w", err)
	}

	return nil
}
