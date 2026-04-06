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

	endSyncChan := make(chan struct{})
	syncCompletedChan := make(chan struct{})
	signalChan := make(chan os.Signal, 1)
	exitChan := make(chan struct{})
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)

	go func() {
		signal := <-signalChan
		switch signal {
		case syscall.SIGUSR1:
			close(endSyncChan)
			notify.Send("Waiting for sync to finish", false)
			<-syncCompletedChan
			notify.Send("Starting full scan", false)
			indexer.RunFullScan()
			isSyncActive = true
			notify.Send("Restarting sync", false)
			syncCompletedChan = make(chan struct{})
			go orchestrateSync(endSyncChan, syncCompletedChan)
		case os.Interrupt, syscall.SIGTERM:
			close(endSyncChan)
			slog.Info("closing down sync processes.")
			<-syncCompletedChan
			slog.Info("exiting program.")
			notify.Send("Service has been stopped", false)
			close(exitChan)
		}
	}()

	endSyncChan = make(chan struct{})
	go orchestrateSync(endSyncChan, syncCompletedChan)

	<-exitChan
	os.Exit(0)
}
