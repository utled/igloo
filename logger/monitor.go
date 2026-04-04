package logger

import (
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"igloo/notify"
)

type LogWriter struct {
	mu                sync.Mutex
	logFilename       string
	logFile           *os.File
	logSize           int64
	archivePath       string
	archiveFilePrefix string
}

// Write implements io.Writer and checks for rotation needs
func (writer *LogWriter) Write(input []byte) (numBytes int, err error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	megabyte := 1024 * 1024
	maxLogSize := 50 * megabyte

	if writer.logSize+int64(len(input)) > int64(maxLogSize) {
		writer.rotate()
	}

	numBytes, err = writer.logFile.Write(input)
	writer.logSize += int64(numBytes)
	return numBytes, err
}

func (writer *LogWriter) rotate() {
	writer.logFile.Close()

	archivedFileName := writer.archiveFilePrefix + time.Now().Format("2006-01-02T15-04-05") + ".log"
	archivedFilePath := filepath.Join(writer.archivePath, archivedFileName)
	_ = os.Rename(writer.logFilename, archivedFilePath)

	newLogFile, err := os.OpenFile(writer.logFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		notify.Send("log rotation has failed.\nManual intervention is required", true)
		panic(err)
	}

	writer.logFile = newLogFile
	writer.logSize = 0

	go writer.zipArchive()
}

func (writer *LogWriter) zipArchive() {
	files, err := os.ReadDir(writer.archivePath)
	if err != nil {
		slog.Error("", "call", "os.ReadDir()", "err", err)
	}
	if len(files) < 6 {
		return
	}

	var countLogFiles, countGzFiles int
	var oldestLogTime, oldestGzTime time.Time
	var oldestLogFile, oldestGzFile os.DirEntry
	for _, file := range files {
		fileName := file.Name()
		switch filepath.Ext(fileName) {
		case ".log":
			countLogFiles++
			info, _ := file.Info()
			if oldestLogTime.IsZero() || info.ModTime().Before(oldestLogTime) {
				oldestLogTime = info.ModTime()
				oldestLogFile = file
			}
		case ".gz":
			countGzFiles++
			info, _ := file.Info()
			if oldestGzTime.IsZero() || info.ModTime().Before(oldestGzTime) {
				oldestGzTime = info.ModTime()
				oldestGzFile = file
			}
		}
	}
	if countLogFiles > 5 {
		oldPath := filepath.Join(writer.archivePath, oldestLogFile.Name())
		zipPath := oldPath + ".gz"

		sourceFile, err := os.Open(oldPath)
		if err != nil {
			slog.Error("", "call", "os.Open()", "err", err)
		}
		defer sourceFile.Close()

		targetFile, err := os.Create(zipPath)
		if err != nil {
			slog.Error("", "call", "os.Create()", "err", err)
		}
		defer targetFile.Close()

		gzWriter := gzip.NewWriter(targetFile)
		_, err = io.Copy(gzWriter, sourceFile)
		if err != nil {
			slog.Error("", "call", "io.Copy()", "err", err)
		}
		gzWriter.Close()

		if err == nil {
			sourceFile.Close()
			os.Remove(oldPath)
			countGzFiles++
		}
	}
	if countGzFiles > 5 {
		pathToRemove := filepath.Join(writer.archivePath, oldestGzFile.Name())
		os.Remove(pathToRemove)
	}
}

// NotificationHandler intercepts log records to detect Errors
type NotificationHandler struct {
	slog.Handler
}

func (notificationHandler *NotificationHandler) Handle(ctx context.Context, logRecord slog.Record) error {
	if logRecord.Level >= slog.LevelError {
		logDetails.lastError = time.Now()
		logDetails.errorCount++

		go notify.Send("New error(s) have been detected.\nCheck the log for details and take action.", true)
	}
	return notificationHandler.Handler.Handle(ctx, logRecord)
}
