// Package config defines config defaults and manages interactions with the config file on disk
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

var excludedFromRoot = []string{
	"/mnt",
	"/boot",
	"/etc",
	"/root",
	"/bin",
	"/sbin",
	"/dev",
	"/sys",
	"/proc",
}

var excludedEntryNames = []string{
	"run",
	"tmp",
	"cache",
}

var contentFileTypes = []string{
	".txt",
	".text",
	".md",
	".csv",
	".doc",
	".docx",
	".odt",

	".go",
	".py",
	".ipynb",
	".sql",
	".c",
	".cpp",
	".h",
	".cs",
	".rs",
	".kt",
	".ktm",
	".kts",
	".java",
	".odin",
	".zig",
	".sh",
	".bash",
	".lua",
	".js",
	".ts",

	".conf",
	".toml",
	".yaml",
	".yml",
	".json",
	".jsonc",
	".xml",
	".dockerfile",
	".html",
	".css",
}

var Details ConfigCollection

type ConfigCollection struct {
	SyncPath               string   `json:"SyncPath"`               // defaults to system root directory
	LogLevel               string   `json:"LogLevel"`               // defaults to "warning"
	ExcludedEntries        []string `json:"ExcludedPaths"`          // specific paths to exclude from being indexed
	ExcludedEntryNames     []string `json:"ExcludedEntryNames"`     // entry names (not full paths) to exclude from being indexed regardless of where found
	HiddenEntriesToInclude []string `json:"HiddenEntriesToInclude"` // all hidden entries are excluded by program logic except for specific paths defined here
	ContentFileTypes       []string `json:"ContentFileTypes"`       // what file types does the index capture the contents for to allow content based searches of the index
	LastModification       time.Time
}

func composeHiddenEntriesToInclude(homePath string) []string {
	hiddenEntriesToInclude := []string{
		filepath.Join(homePath, ".igloo"),
		filepath.Join(homePath, ".local"),
		filepath.Join(homePath, ".config"),
	}

	return hiddenEntriesToInclude
}

func Initialize(homePath string) error {
	servicePath := filepath.Join(homePath, ".igloo")
	configFilepath := filepath.Join(servicePath, "igloo.conf")

	hiddenEntriesToInclude := composeHiddenEntriesToInclude(homePath)

	defaultConfig := ConfigCollection{
		SyncPath:               "/",
		LogLevel:               "warning",
		ExcludedEntries:        excludedFromRoot,
		ExcludedEntryNames:     excludedEntryNames,
		HiddenEntriesToInclude: hiddenEntriesToInclude,
		ContentFileTypes:       contentFileTypes,
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

	return nil
}

func readNewConfig() (newConfig ConfigCollection, err error) {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return newConfig, fmt.Errorf("config.readNewConfig() -> os.UserHomeDir() %w", err)
	}
	configPath := filepath.Join(homePath, ".igloo/igloo.conf")
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return newConfig, fmt.Errorf("config.readNewConfig() -> os.ReadFile() %w", err)
	}

	if err = json.Unmarshal(configFile, &newConfig); err != nil {
		return newConfig, fmt.Errorf("config.readNewConfig() -> json.Unmarshal() %w", err)
	}

	fileStat, err := os.Lstat(configPath)
	if err != nil {
		return newConfig, fmt.Errorf("config.readNewConfig() -> os.Lstat() %w", err)
	}
	newConfig.LastModification = fileStat.ModTime()

	for _, exclusionEntry := range excludedFromRoot {
		if !slices.Contains(newConfig.ExcludedEntries, exclusionEntry) {
			newConfig.ExcludedEntries = append(newConfig.ExcludedEntries, exclusionEntry)
		}
	}

	for _, exclusionEntryName := range excludedEntryNames {
		if !slices.Contains(newConfig.ExcludedEntryNames, exclusionEntryName) {
			newConfig.ExcludedEntryNames = append(newConfig.ExcludedEntryNames, exclusionEntryName)
		}
	}

	return newConfig, nil
}

func Read() error {
	newConfig, err := readNewConfig()
	if err != nil {
		return err
	}
	Details = newConfig

	return nil
}

func CheckUpdate() (isChanged bool, err error) {
	newConfig, err := readNewConfig()
	if err != nil {
		return isChanged, err
	}

	if newConfig.LastModification.After(Details.LastModification) {
		if strings.Compare(Details.SyncPath, newConfig.SyncPath) != 0 ||
			!slices.Equal(Details.ExcludedEntries, newConfig.ExcludedEntries) ||
			!slices.Equal(Details.ExcludedEntryNames, newConfig.ExcludedEntryNames) ||
			!slices.Equal(Details.ContentFileTypes, newConfig.ContentFileTypes) {
			isChanged = true
		}
		Details = newConfig
	}

	return isChanged, nil
}
