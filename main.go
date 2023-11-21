package main

import (
	"log"
	"os"
)

const logFilePath = "/var/log/frc-radio-api.log"

func main() {
	// Set up logging to file.
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	} else {
		log.Printf("error opening log file; logging to stdout instead: %v", err)
	}

	ap := newAccessPoint()
	web := newWeb(ap)
	web.run()
}
