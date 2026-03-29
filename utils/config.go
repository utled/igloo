// Package utils gathers utility functionality not part of the main indexing process
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

var excludedEntries = []string{
	"mnt",
	"boot",
	"etc",
	"root",
	"bin",
	"dev",
	"sys",
	"proc",
	".snapshots",
	".venv",
	".cargo",
	".rustup",
	".git",
	".cache",
	".idea",
}

var entriesToKeep = []string{
	".igloo",
	".local",
	".config",
}

var contentFileTypes = []string{
	".txt",
	".md",
	".go",
	".py",
	".c",
	".cpp",
	".c++",
	".cs",
	".rs",
	".kt",
	".ktm",
	".kts",
	".java",
	".sh",
	".csv",
	".css",
	".lua",
	".dockerfile",
	".json",
	".jsonc",
	".conf",
	".js",
	".ipynb",
	".sql",
	".bash",
	".toml",
	".yaml",
	".yml",
	".xml",
	".ts",
	".doc",
	".docx",
	".docm",
	".xlxs",
	".xlxm",
	".ods",
	".odt",
}

func composeExclusions(homePath string) (exclusions []string, err error) {
	entries, err := os.ReadDir(homePath)
	if err != nil {
		return exclusions, fmt.Errorf("failed to read home directory:%v", err)
	}

	for _, entry := range entries {
		if entry.Name()[0] == '.' && !slices.Contains(entriesToKeep, entry.Name()) {
			exclusions = append(exclusions, entry.Name())
		}
	}

	exclusions = append(exclusions, excludedEntries...)

	return exclusions, nil
}

var Config ConfigDetails

type ConfigDetails struct {
	SyncPath         string   `json:"SyncPath"`         // defaults to system root directory
	WaitBetweenSyncs int      `json:"WaitBetweenSyncs"` // defaults to 1 second
	LogLevel         string   `json:"LogLevel"`         // default to "warning"
	ExcludedEntries  []string `json:"ExcludedEntries"`  // what files and directories are excluded from being indexed
	ContentFileTypes []string `json:"ContentFileTypes"` // what file types does the index capture the contents for to allow content based searches of the index
	LastModification time.Time
}

func InitializeConfig(homePath string) error {
	servicePath := filepath.Join(homePath, ".igloo")
	configFilepath := filepath.Join(servicePath, "igloo.conf")
	if _, err := os.Stat(configFilepath); err != nil {
		exclusions, err := composeExclusions(homePath)
		if err != nil {
			return err
		}

		defaultConfig := ConfigDetails{
			SyncPath:         "/",
			WaitBetweenSyncs: 1,
			LogLevel:         "warning",
			ExcludedEntries:  exclusions,
			ContentFileTypes: contentFileTypes,
		}

		defaultConfigJSON, _ := json.MarshalIndent(defaultConfig, "", "  ")

		file, err := os.Create(configFilepath)
		if err != nil {
			return fmt.Errorf("failed to create config file:\n%v", err)
		}
		defer file.Close()

		_, err = file.WriteString(string(defaultConfigJSON))
		if err != nil {
			return fmt.Errorf("failed to write to config file\n%v", err)
		}
		file.Sync()
	}

	return nil
}

func readConfig() (newConfig ConfigDetails, err error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return newConfig, fmt.Errorf("failed to identify user home directory:%v", err)
	}
	configPath := filepath.Join(homePath, ".igloo/igloo.conf")
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return newConfig, fmt.Errorf("failed to read config file:%v", err)
	}

	if err = json.Unmarshal(configFile, &newConfig); err != nil {
		return newConfig, fmt.Errorf("failed to unmarshal config file:%v", err)
	}

	fileStat, err := os.Lstat(configPath)
	if err != nil {
		return newConfig, fmt.Errorf("failed to read config metadata %w", err)
	}
	newConfig.LastModification = fileStat.ModTime()

	for _, exclusionEntry := range excludedEntries {
		if !slices.Contains(newConfig.ExcludedEntries, exclusionEntry) {
			newConfig.ExcludedEntries = append(newConfig.ExcludedEntries, exclusionEntry)
		}
	}
	return newConfig, nil
}

func GetConfig() error {
	newConfig, err := readConfig()
	if err != nil {
		return err
	}
	Config = newConfig

	return nil
}

func CheckUpdateConfig() (requiresRefresh bool, err error) {
	newConfig, err := readConfig()
	if err != nil {
		return requiresRefresh, err
	}

	if newConfig.LastModification.After(Config.LastModification) {
		if strings.Compare(Config.SyncPath, newConfig.SyncPath) != 0 || !slices.Equal(Config.ExcludedEntries, newConfig.ExcludedEntries) || !slices.Equal(Config.ContentFileTypes, newConfig.ContentFileTypes) {
			requiresRefresh = true
		}
		Config = newConfig
	}

	return requiresRefresh, nil
}
