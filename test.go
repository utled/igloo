package main

import (
	"fmt"
	"igloo/data"
	"igloo/db"
	"os"
	"strconv"
	"syscall"
)

func test() {
	con, err := db.CreateConnection()
	if err != nil {
		fmt.Println("db connection error:", err)
	}
	defer db.CloseConnection(con)

	thePath := "/home/utled/Projects/Go/igloo/main.go"
	fileInfo, err := os.Lstat(thePath)
	if err != nil {
		fmt.Println("lstat error:", err)
	}
	statT := fileInfo.Sys().(*syscall.Stat_t)
	osDevID := statT.Dev
	fmt.Println("osDevID", osDevID)
	osInode := statT.Ino
	fmt.Println("osInode", osInode)

	osUniqueKeyFormatUint := strconv.FormatUint(osDevID, 10) + strconv.FormatUint(osInode, 10) + thePath
	fmt.Println("osUniqueKeyFormatUint", osUniqueKeyFormatUint)
	osUniqueKeyItoa := strconv.Itoa(int(osDevID)) + strconv.Itoa(int(osInode)) + thePath
	fmt.Println("osUniqueKeyItoa", osUniqueKeyItoa)
	indexedEntries, err := data.GetIndexedEntries(con)
	if err != nil {
		fmt.Println("indexed entry error:", err)
	}
	formatUintMatchedEntry, match := indexedEntries[osUniqueKeyFormatUint]
	fmt.Println("found match on FormatUint", match)
	fmt.Println("FormatUint entry:\n", formatUintMatchedEntry)
	itoaMatchedEntry, match := indexedEntries[osUniqueKeyItoa]
	fmt.Println("found match on Itoa", match)
	fmt.Println("Itoa entry:\n", itoaMatchedEntry)
}

