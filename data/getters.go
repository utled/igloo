package data

import (
	"database/sql"
	"fmt"
	"strconv"
)

func GetUniqueMappedEntries(con *sql.DB) (uniqueMappedEntries map[string]EntryHeader, err error) {
	uniqueMappedEntries = make(map[string]EntryHeader)
	var query string
	var response *sql.Rows
	query = `select dev_id, inode, path, modification_time, metadata_change_time 
				from entries
				order by inode;`
	response, err = con.Query(query)
	if err != nil {
		return uniqueMappedEntries, err
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
			return uniqueMappedEntries, fmt.Errorf("failed to serialize entry details to map: %v", err)
		}
		uniqueKey := strconv.Itoa(int(devID)) + strconv.Itoa(int(inode)) + details.Path
		uniqueMappedEntries[uniqueKey] = details
	}
	if err = response.Err(); err != nil {
		return uniqueMappedEntries, fmt.Errorf("failed to iterate through db response: %v", err)
	}

	return uniqueMappedEntries, nil
}
