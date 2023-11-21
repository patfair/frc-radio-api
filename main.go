package main

import (
	"fmt"
	"github.com/digineo/go-uci"
	"log"
	"net/http"
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

	accessPoint := newAccessPoint()
	web := newWeb(accessPoint)
	web.run()
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	sections, _ := uci.GetSections("wireless", "wifi-device")
	_, _ = fmt.Fprintf(w, "%v", sections)
}
