// This file is specific to the access point version of the API.
//go:build !robot

package radio

import (
	"errors"
	"fmt"
	"regexp"
)

const stationSsidRegex = "^[a-zA-Z0-9-]*$"

// ConfigurationRequest represents a JSON request to configure the radio.
type ConfigurationRequest struct {
	// 5GHz or 6GHz channel number for the radio to use. Set to 0 to leave unchanged.
	Channel int `json:"channel"`

	// SSID and WPA key for each team station, keyed by alliance and number (e.g. "red1", "blue3). If a station is not
	// included, its network will be disabled by setting its SSID to a placeholder.
	StationConfigurations map[string]StationConfiguration `json:"stationConfigurations"`
}

// StationConfiguration represents the configuration for a single team station.
type StationConfiguration struct {
	// Team-specific SSID for the station, usually equal to the team number as a string.
	Ssid string `json:"ssid"`

	// Team-specific WPA key for the station. Must be at least eight characters long.
	WpaKey string `json:"wpaKey"`
}

var validLinksysChannels = []int{36, 40, 44, 48, 149, 153, 157, 161, 165}

// Validate checks that all parameters within the configuration request have valid values.
func (request ConfigurationRequest) Validate(radio *Radio) error {
	if request.Channel == 0 && len(request.StationConfigurations) == 0 {
		return errors.New("empty configuration request")
	}

	if request.Channel != 0 {
		// Validate channel number.
		valid := false
		switch radio.Type {
		case typeLinksys:
			for _, channel := range validLinksysChannels {
				if request.Channel == channel {
					valid = true
					break
				}
			}
		case typeVividHosting:
			valid = isValid6GhzChannel(request.Channel)
		}
		if !valid {
			return fmt.Errorf("invalid channel for %s: %d", radio.Type.String(), request.Channel)
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
		if !regexp.MustCompile(stationSsidRegex).MatchString(stationConfiguration.Ssid) {
			return fmt.Errorf("invalid SSID for station %s (expecting alphanumeric with hyphens)", stationName)
		}
		if len(stationConfiguration.WpaKey) < minWpaKeyLength || len(stationConfiguration.WpaKey) > maxWpaKeyLength {
			return fmt.Errorf(
				"invalid WPA key length for station %s: %d (expecting %d-%d)",
				stationName,
				len(stationConfiguration.WpaKey),
				minWpaKeyLength,
				maxWpaKeyLength,
			)
		}
		if !regexp.MustCompile(alphanumericRegex).MatchString(stationConfiguration.WpaKey) {
			return fmt.Errorf("invalid WPA key for station %s (expecting alphanumeric)", stationName)
		}
	}

	return nil
}
