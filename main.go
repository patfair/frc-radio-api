package main

import (
	"github.com/patfair/frc-radio-api/radio"
	"github.com/patfair/frc-radio-api/web"
	"log"
	"os"
)

const logFilePath = "/tmp/frc-radio-api.log"

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

	radio := radio.NewRadio()

	// Launch the web server in a separate thread.
	webServer := web.NewWebServer(radio)
	go webServer.Run()

	// Run the radio event loop in the main thread.
	radio.Run()
}
