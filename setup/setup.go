// Package setup creates file system artifacts (service directory, DB and config file) for the main program to interact with
package setup

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"igloo/config"
	"igloo/db"
	"igloo/logging"
)


func Main() error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("setup.Main() -> os.UserHomeDir() %w", err)
	}

	servicePath := filepath.Join(homePath, ".igloo")

	if info, err := os.Lstat(servicePath); os.IsNotExist(err) {
		slog.Debug("setup process started")

		os.MkdirAll(servicePath, os.ModePerm)
		db.InitializeDB(servicePath)
		config.InitializeConfig(homePath, servicePath)
		logging.InitializeLogger(servicePath)

		slog.Debug("setup process completed")
	} else if err != nil {
		return fmt.Errorf("setup.Main() -> os.Lstat() servicepath %s already exist %w", servicePath, err)
	} else if !info.IsDir() {
		return fmt.Errorf("setup.Main() -> !info.IsDir() servicepath %s is not a directory %v", servicePath, info.Name())
	}

	return nil
}
