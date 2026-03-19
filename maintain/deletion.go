package maintain

import (
	"database/sql"
	"fmt"
	"os"
	"igloo/data"
	"sync"
)

func checkDelete(entryPath string, con *sql.DB) error {
	if _, err := os.Stat(entryPath); err != nil {
		err = data.DeleteEntry(con, entryPath)
		if err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

func traverseIndexedEntries(deletionJobs chan<- string, uniqueMappedEntries map[string]data.EntryHeader, wg *sync.WaitGroup) error {
	defer wg.Done()
	defer close(deletionJobs)

	for _, values := range uniqueMappedEntries {
		deletionJobs <- values.Path
	}
	return nil
}
