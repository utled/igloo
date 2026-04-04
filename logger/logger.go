// Package logger manages setup of log settings and updates of log level
package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"igloo/config"
)

var logDetails = logCollection{
	loglevel: &slog.LevelVar{},
}

type logCollection struct {
	loglevel    *slog.LevelVar
	logFilePath string
	lastError   time.Time
	errorCount  int
}

func Initialize(homeDir string) {
	logFilePath := filepath.Join(homeDir, ".igloo/igloo.log")

	err := os.MkdirAll(filepath.Dir(logFilePath), 0o755)
	if err != nil {
		panic("failed to create log directory: " + err.Error())
	}

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic("failed to open log file: " + err.Error())
	}

	fileInfo, _ := logFile.Stat()
	archivePath := filepath.Join(filepath.Dir(logFilePath), "log_archive")
	archiveFilePrefix := "igloo_"

	rotator := &LogWriter{
		logFilename:       logFilePath,
		logFile:           logFile,
		logSize:           fileInfo.Size(),
		archivePath:       archivePath,
		archiveFilePrefix: archiveFilePrefix,
	}

	logDetails.logFilePath = logFilePath

	logOutPuts := io.MultiWriter(os.Stdout, rotator)

	logDetails.loglevel.Set(slog.LevelInfo)
	logOptions := &slog.HandlerOptions{
		AddSource: true,
		Level:     logDetails.loglevel,
	}

	baseHandler := slog.NewJSONHandler(logOutPuts, logOptions)

	handler := &NotificationHandler{Handler: baseHandler}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("logger initialized", "call", "slog.SetDefault()", "path", logFilePath)

	CheckUpdateLogLevel()
}

func CheckUpdateLogLevel() {
	requestedLevel := config.Details.LogLevel

	switch requestedLevel {
	case "debug":
		if logDetails.loglevel.Level() != slog.LevelDebug {
			slog.Debug("Switching to DEBUG mode")
			logDetails.loglevel.Set(slog.LevelDebug)
		}
	case "info":
		if logDetails.loglevel.Level() != slog.LevelInfo {
			logDetails.loglevel.Set(slog.LevelInfo)
		}
	case "warning":
		if logDetails.loglevel.Level() != slog.LevelWarn {
			slog.Warn("Switching to WARNING mode")
			logDetails.loglevel.Set(slog.LevelWarn)
		}
	case "error":
		if logDetails.loglevel.Level() != slog.LevelError {
			slog.Error("Switching to ERROR mode")
			logDetails.loglevel.Set(slog.LevelError)
		}
	}
}
