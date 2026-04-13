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

func checkIndexCount() (indexCount int, err error) {
	con, err := db.CreateConnection()
	if err != nil {
		return indexCount, err
	}

	query := `select count(*) from entries;`
	response := con.QueryRow(query)
	response.Scan(&indexCount)

	return indexCount, nil
}

func runSetup(homeDir string) error {
	servicePath := filepath.Join(homeDir, ".igloo")

	if _, err := os.Lstat(servicePath); os.IsNotExist(err) {
		os.MkdirAll(servicePath, os.ModePerm)
		os.MkdirAll(filepath.Join(servicePath, "log_archive"), os.ModePerm)
		os.MkdirAll(filepath.Join(servicePath, "tmp"), os.ModePerm)
		if err = db.InitializeDB(servicePath); err != nil { return err }
		if err = config.Initialize(homeDir); err != nil { return err }

		return nil
	}

	if _, err := os.Lstat(filepath.Join(servicePath, "log_archive")); os.IsNotExist(err) {
		os.MkdirAll(filepath.Join(servicePath, "log_archive"), os.ModePerm)
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
	test := flag.Bool("test", false, "run some code")

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
				os.Exit(0)
			}

			err = runSetup(homeDir)
			if err != nil {
				panic(err)
			}

			indexCount, err := checkIndexCount()
			if err != nil {
				panic(err)
			}

			logger.Initialize(homeDir)
			err = config.Read()
			if err != nil {
				panic(err)
			}

			if indexCount <= 0 {
				err := indexer.RunFullScan()
				if err != nil {
					panic(err)
				}
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
		case *test:
			// Do something
		default:
			fmt.Println("unknown command. available commands via 'igloo --help'")
			os.Exit(0)
		}
	}
}
