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
	dirJobs chan<- data.ReadJob,
	fileJobs chan<- data.ReadJob,
	wg *sync.WaitGroup,
	config *data.Config,
) {
	defer wg.Done()

	defer close(dirJobs)
	defer close(fileJobs)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if path == root {
			return nil
		}

		entryStat, err := os.Lstat(path)
		if err != nil {
			return nil
		}

		isDir := d.IsDir()

		if isDir && slices.Contains(config.ExcludedEntries, filepath.Base(path)) {
			return filepath.SkipDir
		}

		readJob := data.ReadJob{Path: path, Stat: &entryStat}
		if isDir {
			dirJobs <- readJob
		} else {
			fileJobs <- readJob
		}

		return nil
	})

	if err != nil {
		log.Printf("Fatal error during directory traversal: %v", err)
	}
}
