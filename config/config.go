// Package config defines the default configurations for the program
// and provides initialization and getters for the externally hosted config file
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"igloo/data"
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

func InitializeConfig(homePath string) error {
	servicePath := filepath.Join(homePath, ".igloo")
	configFilepath := filepath.Join(servicePath, "igloo.conf")
	if _, err := os.Stat(configFilepath); err != nil {
		exclusions, err := composeExclusions(homePath)
		if err != nil {
			return err
		}

		defaultConfig := data.Config{
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

func GetConfig() (config data.Config, err error) {
	config = data.Config{}
	homePath, err := os.UserHomeDir()
	if err != nil {
		return config, fmt.Errorf("failed to identify user home directory:%v", err)
	}
	configPath := filepath.Join(homePath, ".igloo/igloo.conf")
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file:%v", err)
	}

	if err = json.Unmarshal(configFile, &config); err != nil {
		return config, fmt.Errorf("failed to unmarshal config file:%v", err)
	}

	for _, exclusionEntry := range excludedEntries {
		if !slices.Contains(config.ExcludedEntries, exclusionEntry) {
			config.ExcludedEntries = append(config.ExcludedEntries, exclusionEntry)
		}
	}

	return config, nil
}
