// Package maintain manages the continous syncing and updating/deletion of indexed file system entries
package maintain

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"igloo/initial"
	"igloo/utils"
)

func recordPID(pidPath string) error {
	pid := os.Getpid()
	err := os.WriteFile(pidPath, fmt.Appendf(nil, "%d", pid), 0o644)
	//err :=	os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", pid)), 0o644)
	if err != nil {
		return fmt.Errorf("maintain.recordPID() -> os.WriteFile() %w", err)
	}
	return nil
}

func StartIndexSync(homeDir string) {
	pidPath := filepath.Join(homeDir, ".igloo/tmp/igloo.pid")
	err := recordPID(pidPath)
	if err != nil {
		slog.Error("failed to record process ID", "call", "recordPID()", "err", err)
		os.Exit(1)
	}

	isSyncActive := true
	syncChan := make(chan struct{})
	go orchestrateSync(&isSyncActive, syncChan)

	signalChan := make(chan os.Signal, 1)
	exitChan := make(chan struct{})
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)

	go func() {
		signal := <- signalChan
		switch signal {
		case syscall.SIGUSR1:
			isSyncActive = false
			utils.Notify("Waiting for sync to finish", false)
			<-syncChan
			utils.Notify("Starting full scan", false)
			initial.StartInitialScan()
			isSyncActive = true
			utils.Notify("Restarting sync", false)
			syncChan = make(chan struct{})
			go orchestrateSync(&isSyncActive, syncChan)
		case os.Interrupt, syscall.SIGTERM:
			isSyncActive = false
			slog.Info("closing down sync processes.")
			<-syncChan
			slog.Info("exiting program.")
			utils.Notify("Service has been stopped", false)
			close(exitChan)
		}
	}()

	<-exitChan
	os.Exit(0)
}
