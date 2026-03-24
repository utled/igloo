package main

import (
	"bufio"
	"fmt"
	"igloo/initial"
	"igloo/maintain"
	"igloo/setup"
	"os"
	"strings"
)

func main() {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		arguments := strings.Split(strings.TrimSpace(input), " ")
		switch arguments[0] {
		case "test":
			//test()			
		case "setup":
			err := setup.Main()
			if err != nil {
				fmt.Println(err)
			}
		case "fullscan":
			initial.StartInitialScan()
		case "sync":
			maintain.StartIndexSync()
		case "exit":
			os.Exit(0)
		default:
			fmt.Println(arguments)
		}
	}

}
