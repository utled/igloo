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
		return err
	}
	defer CloseConnection(db)

	err = createTables(db)
	if err != nil {
		return err
	}

	return nil
}

func createTables(db *sql.DB) error {
	tableStatements := []string{
		`create table if not exists full_scans (
    		scan_id integer primary key,
         	scan_start text,
         	scan_end text,
         	scan_duration text,
         	directory_count int,
         	file_count int,
         	file_w_content_count int,
         	ignored_entries_count int,
         	indexing_completed bool
         );`,
		`create table if not exists entries (
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
    		filetype text,
    		content_snippet text,
    		full_text text,
    		line_count_total int,
    		line_count_w_content int,
				primary key(dev_id, inode, path)
		) without rowid;`,
	}

	for _, statement := range tableStatements {
		_, err := db.Exec(statement)
		if err != nil {
			return fmt.Errorf("could not create table %s: \n%w", statement, err)
		}
	}

	return nil
}

func CreateConnection() (*sql.DB, error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(homePath, ".igloo", "igloo.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	return db, nil
}

func CloseConnection(db *sql.DB) error {
	err := db.Close()
	if err != nil {
		return fmt.Errorf("faIled to close db connection: %v", err)
	}

	return nil
}
