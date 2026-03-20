package data

import (
	"database/sql"
	"fmt"
	"strconv"
)

func GetIndexedEntries(con *sql.DB) (indexedEntries map[string]EntryHeader, err error) {
	indexedEntries = make(map[string]EntryHeader)
	var query string
	var response *sql.Rows
	query = `select dev_id, inode, path, modification_time, metadata_change_time 
				from entries
				order by inode;`
	response, err = con.Query(query)
	if err != nil {
		return indexedEntries, err
	}

	for response.Next() {
		var devID uint64
		var inode uint64
		var details EntryHeader
		err = response.Scan(
			&devID,
			&inode,
			&details.Path,
			&details.ModificationTime,
			&details.MetaDataChangeTime,
		)
		if err != nil {
			return indexedEntries, fmt.Errorf("failed to serialize entry details to map: %v", err)
		}
		uniqueKey := strconv.Itoa(int(devID)) + strconv.Itoa(int(inode)) + details.Path
		indexedEntries[uniqueKey] = details
	}
	if err = response.Err(); err != nil {
		return indexedEntries, fmt.Errorf("failed to iterate through db response: %v", err)
	}

	return indexedEntries, nil
}
