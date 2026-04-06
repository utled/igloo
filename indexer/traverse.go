package indexer

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"igloo/config"
)

// traverseDirectory crawls the whole filesystem tree from the top level path defined in the config file
// it excludes the paths defined as out of scope (root, boot, proc et.c. + user defined exclusions)
// before producing jobs to the readJob channel to start collecting entry details
func traverseDirectory(
	root string,
	readJobs chan<- readJob,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	defer close(readJobs)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if !os.IsPermission(err) {
				slog.Error(fmt.Sprintf("failed to walk path %s", path), "call", "filepath.Walkdir()", "err", err)
			}
			return nil
		}

		entryStat, err := os.Lstat(path)
		if err != nil {
			slog.Error(fmt.Sprintf("failed on path %s", path), "call", "filepath.WalkDir() -> os.Lstat()", "err", err)
			return nil
		}

		if d.IsDir() && slices.Contains(config.Details.ExcludedEntries, filepath.Base(path)) {
			return filepath.SkipDir
		}

		readJobs <- readJob{path: path, stat: &entryStat}


		return nil
	})
	if err != nil {
		slog.Error("failed to walk directory", "call", "filepath.WalkDir()", "err", err)
	}
}
