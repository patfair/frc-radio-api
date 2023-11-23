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

var validLinksysChannels = []int{36, 40, 44, 48, 149, 153, 157, 161, 165}

// configurationHandler receives a JSON request to configure the access point and adds it to the asynchronous queue.
func (web *web) configurationHandler(w http.ResponseWriter, r *http.Request) {
	var request configurationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		errorMessage := "Error: invalid JSON: " + err.Error()
		log.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	if reflect.DeepEqual(request, configurationRequest{}) {
		errorMessage := "Error: received empty configuration request"
		log.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	if err := request.validate(web.accessPoint.hardwareType); err != nil {
		errorMessage := "Error: " + err.Error()
		log.Println(errorMessage)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	log.Printf("Received configuration request: %+v", request)
	web.accessPoint.enqueueConfigurationRequest(request)
	_, _ = fmt.Fprintln(w, "New configuration received and will be applied asynchronously.")
}

// validate checks that all parameters within the configuration request have valid values.
func (request configurationRequest) validate(accessPointType accessPointType) error {
	if request.Channel != 0 {
		// Validate channel number.
		valid := false
		switch accessPointType {
		case typeLinksys:
			for _, channel := range validLinksysChannels {
				if request.Channel == channel {
					valid = true
					break
				}
			}
		case typeVividHosting:
			x := (request.Channel - 5) / 8
			y := (request.Channel - 5) % 8
			valid = y == 0 && x >= 0 && x <= 28
		}
		if !valid {
			return fmt.Errorf("invalid channel for %s: %d", accessPointType.String(), request.Channel)
		}
	}

	// Validate station configurations.
	for stationName, stationConfiguration := range request.StationConfigurations {
		stationNameValid := false
		for name := red1; name < stationCount; name++ {
			if stationName == name.String() {
				stationNameValid = true
				break
			}
		}
		if !stationNameValid {
			return fmt.Errorf("invalid station: %s", stationName)
		}
		if stationConfiguration.Ssid == "" {
			return fmt.Errorf("SSID for station %s cannot be blank", stationName)
		}
		if len(stationConfiguration.WpaKey) < 8 {
			return fmt.Errorf(
				"invalid WPA key length for station %s: %d", stationName, len(stationConfiguration.WpaKey),
			)
		}
	}

	return nil
}
