package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"igloo/initial"
	"igloo/maintain"
	"igloo/setup"
)

func main() {
	err := setup.Main()
	if err != nil {
		panic(err)
	}
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		arguments := strings.Split(strings.TrimSpace(input), " ")
		switch arguments[0] {
		case "test":
			// test()
		case "init":
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
