// Package logger manages setup of log settings and updates of log level
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"igloo/config"
)

var LogLevel = &slog.LevelVar{}

func Initialize(homeDir string) {
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

	CheckUpdateLogLevel()
}

func CheckUpdateLogLevel() {
	switch config.Details.LogLevel {
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
