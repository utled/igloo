// Package logging defines the default logger for the program
package logging

import (
	"igloo/config"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var LogLevel = &slog.LevelVar{}

func InitializeLogger(homeDir string) {
	logFilePath := filepath.Join(homeDir, ".igloo/igloo.log")
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		file, err := os.Create(logFilePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()
	}

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(err)
	}
	logOutPuts := io.MultiWriter(os.Stdout, logFile)

	LogLevel.Set(slog.LevelInfo)
	logOptions := &slog.HandlerOptions{
		AddSource: true,
		Level:     LogLevel,
	}

	handler := slog.NewJSONHandler(logOutPuts, logOptions)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	slog.Info("logger initialized", "call", "slog.SetDefault()")


	config, err := config.GetConfig()
	if err != nil {
		slog.Error("", "call", "config.GetConfig()", "err", err)
	}
	ChangeLogLevel(config.LogLevel)
}

func ChangeLogLevel(logLevel string) {
	switch logLevel {
	case "debug":
		if LogLevel.Level() != slog.LevelDebug {
			slog.Info("Switching to DEBUG mode")
			LogLevel.Set(slog.LevelDebug)
		}
	case "info":
		if LogLevel.Level() != slog.LevelInfo {
			slog.Info("Switching to INFO mode")
			LogLevel.Set(slog.LevelInfo)
		}
	case "warning":
		if LogLevel.Level() != slog.LevelWarn {
			slog.Info("Switching to WARNING mode")
			LogLevel.Set(slog.LevelWarn)
		}
	case "error":
		if LogLevel.Level() != slog.LevelError {
			slog.Info("Switching to ERROR mode")
			LogLevel.Set(slog.LevelError)
		}
	}
}
