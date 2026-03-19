package main

import (
	"fmt"
	"os"
	"syscall"
)

func test() {
	//thePath := "/home/utled/Projects/Py/DataScienceAndMachineLearning/.venv/bin/python"
	thePath := "/home/utled/Projects/Go/igloo/main.go"
	fileInfo, err := os.Stat(thePath)
	if err != nil {
		fmt.Println("stat error:", err)
	}
	fmt.Println("stat:", fileInfo)
	statT := fileInfo.Sys().(*syscall.Stat_t)
	
	fmt.Println("stat_t:", statT)
	/*
	entries, err := os.ReadDir("/home/utled/Projects/Py/DataScienceAndMachineLearning/.venv/bin")
	if err != nil {
		fmt.Println(err)
	}

	for _, entry := range entries {
		// Check if the entry is a symbolic link
		if entry.Type()&os.ModeSymlink != 0 {
			fmt.Printf("%s is a symlink\n", entry.Name())
		} else {
			fmt.Printf("%s is a regular file or directory\n", entry.Name())
		}
	}
	*/
}

