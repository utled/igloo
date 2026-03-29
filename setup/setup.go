// Package setup creates file system artifacts (service directories, DB and config file) for the main program to interact with
package setup

import (
	"os"
	"path/filepath"

	"igloo/config"
	"igloo/db"
)

func checkSetupStatus(homeDir string) (needsSetup bool, err error) {
	servicePath := filepath.Join(homeDir, ".igloo")

	var relevantPaths []string
	relevantPaths = append(relevantPaths, servicePath)
	relevantPaths = append(relevantPaths, filepath.Join(servicePath, "tmp"))
	relevantPaths = append(relevantPaths, filepath.Join(servicePath, "igloo.db"))
	relevantPaths = append(relevantPaths, filepath.Join(servicePath, "igloo.conf"))
	for _, path := range relevantPaths {
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			needsSetup = true
			return needsSetup, nil
		}
	}

	return needsSetup, nil
}

func RunSetup(homeDir string) error {
	servicePath := filepath.Join(homeDir, ".igloo")
	needsSetup, err := checkSetupStatus(homeDir)
	if err != nil {
		return err
	}
	if !needsSetup {
		return nil
	}

	if _, err := os.Lstat(servicePath); os.IsNotExist(err) {
		os.MkdirAll(servicePath, os.ModePerm)
		os.MkdirAll(filepath.Join(servicePath, "tmp"), os.ModePerm)
		db.InitializeDB(servicePath)
		config.InitializeConfig(homeDir)

		return nil
	}

	if _, err := os.Lstat(filepath.Join(servicePath, "tmp")); os.IsNotExist(err) {
		os.MkdirAll(filepath.Join(servicePath, "tmp"), os.ModePerm)
	}
	if _, err := os.Lstat(filepath.Join(servicePath, "igloo.db")); os.IsNotExist(err) {
		db.InitializeDB(servicePath)
	}
	if _, err := os.Lstat(filepath.Join(servicePath, "igloo.conf")); os.IsNotExist(err) {
		config.InitializeConfig(homeDir)
	}

	return nil
}
