// Package maintain manages the continous syncing and updating/deletion of indexed file system entries
package maintain

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"igloo/notifications"
)

func StartIndexSync() {
	isSyncActive := true
	syncChan := make(chan struct{})
	go orchestrateSync(&isSyncActive, syncChan)

	signalChan := make(chan os.Signal, 1)
	exitChan := make(chan struct{})
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)

	go func() {
		<-signalChan
		isSyncActive = false
		slog.Info("closing down sync processes.")
		<-syncChan
		slog.Info("exiting program.")
		notifications.Notify("Service has been stopped", false)
		close(exitChan)
	}()

	<-exitChan
	os.Exit(0)
}
