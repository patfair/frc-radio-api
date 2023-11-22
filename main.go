package main

import (
	"log"
	"os"
)

const logFilePath = "/root/frc-radio-api.log"

func main() {
	// Set up logging to file.
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	} else {
		log.Printf("error opening log file; logging to stdout instead: %v", err)
	}
	log.Println("Starting FRC Radio API...")

	ap := newAccessPoint()

	// Launch the web server in a separate thread.
	web := newWeb(ap)
	go web.run()

	// Run the access point loop in the main thread.
	ap.run()
}
