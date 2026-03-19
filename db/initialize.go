package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
)

type DefaultConfig struct{}

func InitializeDB(servicePath string) error {
	dbPath := filepath.Join(servicePath, "igloo.db")
	db, err := CreateConnection(dbPath)
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
		`create table if not exists tagged_entries (
    		inode int not null primary key,
    		tags text
		);`,
		`create table if not exists ignored_entries (
    		path text,
    		error text
		);`,
	}

	for _, statement := range tableStatements {
		_, err := db.Exec(statement)
		if err != nil {
			return fmt.Errorf("could not create table %s: \n%w", statement, err)
		}
	}

	return nil
}
