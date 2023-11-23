package main

import (
	"github.com/patfair/frc-radio-api/radio"
	"github.com/patfair/frc-radio-api/web"
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

	ap := radio.NewAccessPoint()

	// Launch the web server in a separate thread.
	webServer := web.NewWebServer(ap)
	go webServer.Run()

	// Run the access point loop in the main thread.
	ap.Run()
}
