package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"igloo/config"
	"igloo/db"
	"igloo/indexer"
	"igloo/logger"
	"igloo/syncer"
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

func checkSetupStatus(homeDir string) (needsSetup bool, err error) {
	servicePath := filepath.Join(homeDir, ".igloo")

	var relevantPaths []string
	relevantPaths = append(relevantPaths, servicePath)
	relevantPaths = append(relevantPaths, filepath.Join(servicePath, "tmp"))
	relevantPaths = append(relevantPaths, filepath.Join(servicePath, "igloo.db"))
	relevantPaths = append(relevantPaths, filepath.Join(servicePath, "igloo.conf"))
	for _, path := range relevantPaths {
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			needsSetup = true
			return needsSetup, nil
		}
	}

	return needsSetup, nil
}

func runSetup(homeDir string) error {
	servicePath := filepath.Join(homeDir, ".igloo")
	needsSetup, err := checkSetupStatus(homeDir)
	if err != nil {
		return err
	}
	if !needsSetup {
		return nil
	}

	if _, err := os.Lstat(servicePath); os.IsNotExist(err) {
		os.MkdirAll(servicePath, os.ModePerm)
		os.MkdirAll(filepath.Join(servicePath, "tmp"), os.ModePerm)
		db.InitializeDB(servicePath)
		config.Initialize(homeDir)

		return nil
	}

	if _, err := os.Lstat(filepath.Join(servicePath, "tmp")); os.IsNotExist(err) {
		os.MkdirAll(filepath.Join(servicePath, "tmp"), os.ModePerm)
	}
	if _, err := os.Lstat(filepath.Join(servicePath, "igloo.db")); os.IsNotExist(err) {
		db.InitializeDB(servicePath)
	}
	if _, err := os.Lstat(filepath.Join(servicePath, "igloo.conf")); os.IsNotExist(err) {
		config.Initialize(homeDir)
	}

	return nil
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
			err := runSetup(homeDir)
			if err != nil {
				panic(err)
			}
			fmt.Println("setup completed")
		case *start:
			if details.isOngoing {
				fmt.Println("an instance of igloo is already running. call 'igloo --stop' to terminate the process")
				return
			}
			logger.Initialize(homeDir)
			err := indexer.StartFullScan()
			if err != nil {
				panic(err)
			}
			syncer.StartIndexSync(homeDir)
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
