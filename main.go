package main

import (
	"fmt"
	"github.com/patfair/frc-radio-api/radio"
	"github.com/patfair/frc-radio-api/web"
	"log"
	"os"
)

const (
	// Path of the current log file.
	logFilePath = "/root/frc-radio-api.log"

	// Path of the old log file, which is rotated when the current log file gets too big.
	oldLogFilePath = "/root/frc-radio-api.log.old"

	// Maximum size of the current log file in bytes.
	logFileMaxSizeBytes = 3 * 1 << 19 // 1.5 MB
)

func main() {
	setupLogging()

	radio := radio.NewRadio()
	fmt.Println("created radio")

	// Launch the web server in a separate thread.
	webServer := web.NewWebServer(radio)
	fmt.Println("created webserver")
	go webServer.Run()

	// Run the radio event loop in the main thread.
	radio.Run()
}

// setupLogging sets up logging to a file, or to stdout if the file can't be opened.
func setupLogging() {
	// Rotate the log file if the current one is too big.
	if fileInfo, err := os.Stat(logFilePath); err == nil {
		if fileInfo.Size() >= logFileMaxSizeBytes {
			if err := os.Rename(logFilePath, oldLogFilePath); err != nil {
				log.Printf("error rotating log file: %v", err)
			}
		}
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	} else {
		log.Printf("error opening log file; logging to stdout instead: %v", err)
	}
	log.Println("Starting FRC Radio API...")
}
