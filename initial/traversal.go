package initial

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"igloo/data"
	"sync"
)

func traverseDirectory(
	root string,
	dirJobs chan<- string,
	fileJobs chan<- string,
	wg *sync.WaitGroup,
	theWorks *data.CollectedInfo,
	config *data.Config,
) {
	defer wg.Done()

	defer close(dirJobs)
	defer close(fileJobs)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			failedPath := data.NotAccessedPaths{Path: path, Err: err.Error()}
			theWorks.Mu.Lock()
			theWorks.NumOfIgnoredEntries += 1
			theWorks.NotRegistered = append(theWorks.NotRegistered, &failedPath)
			theWorks.Mu.Unlock()
			return nil
		}

		if path == root {
			return nil
		}

		_, err = os.Stat(path)
		if err != nil {
			return nil
		}

		if d.IsDir() && slices.Contains(config.ExcludedEntries, filepath.Base(path)) {
			return filepath.SkipDir
		}

		if d.IsDir() {
			dirJobs <- path
		} else {
			fileJobs <- path
		}

		return nil
	})

	if err != nil {
		log.Printf("Fatal error during directory traversal: %v", err)
	}
}
