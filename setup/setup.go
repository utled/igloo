// Package setup creates file system artifacts (service directory, DB and config file) for the main program to interact with
package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"igloo/config"
	"igloo/db"
)


func Main() error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to identify user home directory:%v", err)
	}

	servicePath := filepath.Join(homePath, ".igloo")
	fmt.Println("servicePath:", servicePath)

	if info, err := os.Lstat(servicePath); os.IsNotExist(err) {
		fmt.Println("starting setup process")

		os.MkdirAll(servicePath, os.ModePerm)
		db.InitializeDB(servicePath)
		config.InitializeConfig(homePath, servicePath)

		fmt.Println("setup complete")
	} else if err != nil {
		return fmt.Errorf("conflicting service path was found. path already exist: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("conflicting service path was found. path exist but is not a directory%v", info.Name())
	}

	return nil
}
