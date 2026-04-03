// Package syncer manages the continous syncing and updating/deletion of indexed file system entries
package syncer

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"igloo/indexer"
	"igloo/notify"
)

// recordPID writes the current process ID to tmp file on disk
// to enable checking for already ongoing processes before starting new
// and enabling os signals to an ongoing process
func recordPID(pidPath string) error {
	pid := os.Getpid()
	err := os.WriteFile(pidPath, fmt.Appendf(nil, "%d", pid), 0o644)
	if err != nil {
		return fmt.Errorf("syncer.recordPID() -> os.WriteFile() %w", err)
	}
	return nil
}

var isSyncActive bool

// StartIndexSync is the entry point for the syncing process
// sets up signal listener to enable outside communication to process to trigger full refresh and mange graceful termination
// before triggering sync process orchestration
func StartIndexSync(homeDir string) {
	pidPath := filepath.Join(homeDir, ".igloo/tmp/igloo.pid")
	err := recordPID(pidPath)
	if err != nil {
		slog.Error("failed to record process ID", "call", "recordPID()", "err", err)
		os.Exit(1)
	}

	syncChan := make(chan struct{})
	signalChan := make(chan os.Signal, 1)
	exitChan := make(chan struct{})
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)

	go func() {
		signal := <-signalChan
		switch signal {
		case syscall.SIGUSR1:
			isSyncActive = false
			notify.Send("Waiting for sync to finish", false)
			<-syncChan
			notify.Send("Starting full scan", false)
			indexer.StartFullScan()
			isSyncActive = true
			notify.Send("Restarting sync", false)
			syncChan = make(chan struct{})
			go orchestrateSync(&isSyncActive, syncChan)
		case os.Interrupt, syscall.SIGTERM:
			isSyncActive = false
			slog.Info("closing down sync processes.")
			<-syncChan
			slog.Info("exiting program.")
			notify.Send("Service has been stopped", false)
			close(exitChan)
		}
	}()

	isSyncActive = true
	go orchestrateSync(&isSyncActive, syncChan)

	<-exitChan
	os.Exit(0)
}
