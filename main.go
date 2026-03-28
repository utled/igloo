package main

import (
	"flag"
	"fmt"

	"igloo/initial"
	"igloo/maintain"
	"igloo/notifications"
	"igloo/setup"
)

func main() {
	notify := flag.Bool("notify", false, "for testing purposes. triggers a meaningless dbus notification")
	setupOnly := flag.Bool("setup", false, "runs the setup/initialization of db, config file et.c without starting the indexing process")
	init := flag.Bool("init", false, "starts an initial scan of the whole file system (runs setup steps if not run seperately)")
	sync := flag.Bool("sync", false, "starts the continuous index sync")
	refresh := flag.Bool("refresh", false, "sends USR1 signal to an ongoing sync process to run a full index refresh before resuming sync")

	flag.Parse()
	if flag.NFlag() > 0 {
		switch {
		case *setupOnly:
			err := setup.Main()
			if err != nil {
				panic(err)
			}
		case *init:
			err := setup.Main()
			if err != nil {
				panic(err)
			}
			initial.StartInitialScan()
		case *sync:
			maintain.StartIndexSync()
		case *refresh:
			// signal already running PID
		case *notify:
			notifications.Notify("a notification", false)
		default:
			fmt.Println("unknown command. available commands via --help")
		}
	}

	fmt.Println("requires command. available commands via --help")
}
