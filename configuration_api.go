package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
)

// configurationRequest represents a JSON request to configure the access point.
type configurationRequest struct {
	Channel               int                             `json:"channel"`
	StationConfigurations map[string]stationConfiguration `json:"stationConfigurations"`
}

// stationConfiguration represents the configuration for a single team station.
type stationConfiguration struct {
	Ssid   string `json:"ssid"`
	WpaKey string `json:"wpaKey"`
}

// configurationHandler receives a JSON request to configure the access point and adds it to the asynchronous queue.
func (web *web) configurationHandler(w http.ResponseWriter, r *http.Request) {
	var request configurationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		handleWebErr(w, err)
		return
	}

	if reflect.DeepEqual(request, configurationRequest{}) {
		errorMessage := "Error: received empty configuration request"
		log.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	log.Printf("Received configuration request: %+v", request)
	web.accessPoint.enqueueConfigurationRequest(request)
	_, _ = fmt.Fprintln(w, "New configuration received and will be applied asynchronously.")
}
