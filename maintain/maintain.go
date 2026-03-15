package maintain

import (
	"fmt"
	"igloo/config"
	"time"
)

func Start() error {
	config, err := config.GetConfig()
	if err != nil {
		fmt.Println(err)
	}
	scanCount := 10
	for scanCount > 0 {
		var startPath string
		var err error

		if scanCount%config.LargeSyncFrequenzy == 0 {
			startPath = config.LargeSyncPath
		} else {
			startPath = config.QuickSyncPath
		}

		fmt.Printf("Starting scan of: %s\n", startPath)
		startTime := time.Now()
		err = orchestrateScan(startPath, &config)
		if err != nil {
			return err
		}
		fmt.Println("Scan completed")
		elapsed := time.Since(startTime)
		fmt.Printf("Scan took %s\n", elapsed)

		time.Sleep(1 * time.Second)

		if scanCount == 1 {
			scanCount = 10
		} else {
			scanCount--
		}

	}

	return nil
}
