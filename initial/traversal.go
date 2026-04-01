package initial

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"igloo/data"
	"igloo/utils"
)

func traverseDirectory(
	root string,
	readJobs chan<- data.ReadJob,
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

		if d.IsDir() && slices.Contains(utils.Config.ExcludedEntries, filepath.Base(path)) {
			return filepath.SkipDir
		}

		readJobs <- data.ReadJob{Path: path, Stat: &entryStat}

		return nil
	})
	if err != nil {
		slog.Error("failed to walk directory", "call", "filepath.WalkDir()", "err", err)
	}
}
