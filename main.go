package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"igloo/initial"
	"igloo/utils"
	"igloo/maintain"
)

type ongoingProcessDetails struct {
	pidPath   string
	process   *os.Process
	isOngoing bool
}

func getOngoingProcessDetails(homeDir string) (ongoingProcessDetails, error) {
	details := ongoingProcessDetails{}
	pidPath := filepath.Join(homeDir, ".igloo/tmp/igloo.pid")
	data, _ := os.ReadFile(pidPath)
	pid, _ := strconv.Atoi(string(data))

	details.pidPath = pidPath
	details.process, _ = os.FindProcess(pid)
	err := details.process.Signal(syscall.Signal(0))
	if err == nil {
		details.isOngoing = true
	}

	return details, nil
}

func main() {
	setupOnly := flag.Bool("setup", false, "runs the setup/initialization of db, config file et.c without starting indexing processes")
	start := flag.Bool("start", false, "starts an initial scan of the whole file system (runs setup steps if needed)")
	refresh := flag.Bool("refresh", false, "sends SIGUSR1 to any ongoing sync process to run a full index refresh before resuming sync")
	stop := flag.Bool("stop", false, "(gracefully) terminates any ongoing sync process")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	details, err := getOngoingProcessDetails(homeDir)
	if err != nil {
		panic(err)
	}

	flag.Parse()
	if flag.NFlag() > 0 {
		switch {
		case *setupOnly:
			err := utils.RunSetup(homeDir)
			if err != nil {
				panic(err)
			}
			fmt.Println("setup completed")
		case *start:
			if details.isOngoing {
				fmt.Println("an instance of igloo is already running. call 'igloo --stop' to terminate the process")
				return
			}
			utils.InitializeLogger(homeDir)
			initial.StartInitialScan()
			maintain.StartIndexSync(homeDir)
		case *refresh:
			if details.isOngoing {
				details.process.Signal(syscall.SIGUSR1)
			} else {
				fmt.Println("no ongoing igloo process found")
				os.Exit(0)
			}
		case *stop:
			if details.isOngoing {
				details.process.Signal(syscall.SIGTERM)
			} else {
				fmt.Println("no ongoing igloo process found")
				os.Exit(0)
			}
		default:
			fmt.Println("unknown command. available commands via 'igloo --help'")
			os.Exit(0)
		}
	}
}
