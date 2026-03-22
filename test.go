package main

import (
	"fmt"
	"os"
)

func test() {
	//thePath := "/home/utled/Projects/Py/DataScienceAndMachineLearning/.venv/bin/python"
	thePath := "/home/utled/.local/share/Steam/steamrt64/steam-runtime-steamrt/var/tmp-ELG3D3/usr/share/X11/rgb.txt"
	fileInfo, err := os.Lstat(thePath)
	if err != nil {
		fmt.Println("lstat error:", err)
	}
	if fileInfo.Mode().Type()&os.ModeSymlink != 0 {
		fmt.Println("is a symlink")
	}
	fmt.Println("lstat:", fileInfo)
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

