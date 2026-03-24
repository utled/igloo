// Package maintain manages the continous syncing and updating/deletion of indexed file system entries
package maintain

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func StartIndexSync() {
	isSyncActive := true
	syncChan := make(chan struct{})
	go orchestrateSync(&isSyncActive, syncChan)

	signalChan := make(chan os.Signal, 1)
	exitChan := make(chan struct{})
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		isSyncActive = false
		fmt.Println("\nClosing down sync processes. Please wait...")
		<-syncChan
		close(exitChan)
	}()

	<-exitChan
	os.Exit(0)
}
