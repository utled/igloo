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
	dirJobs chan<- data.ReadJob,
	fileJobs chan<- data.ReadJob,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	defer close(dirJobs)
	defer close(fileJobs)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				slog.Debug("", "call", "filepath.WalkDir() -> os.IsPermission()", "err", err)
			} else {
				slog.Error(fmt.Sprintf("failed to walk path %s", path), "call", "filepath.Walkdir()", "err", err)
			}
			return nil
		}

		if path == root {
			return nil
		}

		entryStat, err := os.Lstat(path)
		if err != nil {
			slog.Error(fmt.Sprintf("failed on path %s", path), "call", "filepath.WalkDir() -> os.Lstat()", "err", err)
			return nil
		}

		isDir := d.IsDir()

		if isDir && slices.Contains(utils.Config.ExcludedEntries, filepath.Base(path)) {
			slog.Debug(fmt.Sprintf("excluded path %s", path), "call", "filepath.WalkDir() -> isDir && slices.Contains()")
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
		slog.Error("failed to walk directory", "call", "filepath.WalkDir()", "err", err)
	}
}
