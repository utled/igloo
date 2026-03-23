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
	deletionChan := make(chan struct{})
	syncChan := make(chan struct{})
	go manageDeletions(&isSyncActive, deletionChan)
	go manageSync(&isSyncActive, syncChan)

	signalChan := make(chan os.Signal, 1)
	exitChan := make(chan struct{})
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		isSyncActive = false
		fmt.Println("\nClosing down sync processes. Please wait...")
		<-deletionChan
		<-syncChan
		close(exitChan)
	}()

	<-exitChan
	os.Exit(0)
}
