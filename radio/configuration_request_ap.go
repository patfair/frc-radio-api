// This file is specific to the access point version of the API.
//go:build !robot

package radio

import (
	"errors"
	"fmt"
	"regexp"
)

// ConfigurationRequest represents a JSON request to configure the radio.
type ConfigurationRequest struct {
	// 5GHz or 6GHz channel number for the radio to use. Set to 0 to leave unchanged.
	Channel int `json:"channel"`

	// Channel bandwidth mode for the radio to use. Valid values are "20MHz" and "40MHz". Set to an empty string to
	// leave unchanged.
	ChannelBandwidth string `json:"channelBandwidth"`

	// VLANs to use for the teams of the red alliance. Valid values are "10_20_30", "40_50_60", and "70_80_90".
	RedVlans AllianceVlans `json:"redVlans"`

	// VLANs to use for the teams of the blue alliance. Valid values are "10_20_30", "40_50_60", and "70_80_90".
	BlueVlans AllianceVlans `json:"blueVlans"`

	// SSID and WPA key for each team station, keyed by alliance and number (e.g. "red1", "blue3). If a station is not
	// included, its network will be disabled by setting its SSID to a placeholder.
	StationConfigurations map[string]StationConfiguration `json:"stationConfigurations"`

	// IP address of the syslog server to send logs to (via UDP on port 514).
	SyslogIpAddress string `json:"syslogIpAddress"`
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
	if request.Channel == 0 && request.ChannelBandwidth == "" && len(request.StationConfigurations) == 0 &&
		request.RedVlans == "" && request.BlueVlans == "" && request.SyslogIpAddress == "" {
		return errors.New("empty configuration request")
	}

	if request.Channel != 0 {
		// Validate channel number.
		valid := false
		switch radio.Type {
		case TypeLinksys:
			for _, channel := range validLinksysChannels {
				if request.Channel == channel {
					valid = true
					break
				}
			}
		case TypeVividHosting:
			valid = isValid6GhzChannel(request.Channel)
		}
		if !valid {
			return fmt.Errorf("invalid channel for %s: %d", radio.Type.String(), request.Channel)
		}
	}

	if request.ChannelBandwidth != "" {
		// Validate channel bandwidth.
		if radio.Type == TypeLinksys {
			return fmt.Errorf("channel bandwidth cannot be changed on %s", radio.Type.String())
		}
		if request.ChannelBandwidth != "20MHz" && request.ChannelBandwidth != "40MHz" {
			return fmt.Errorf("invalid channel bandwidth: %s", request.ChannelBandwidth)
		}
	}

	if request.RedVlans != "" || request.BlueVlans != "" {
		if request.RedVlans == "" || request.BlueVlans == "" {
			return errors.New("both red and blue VLANs must be specified")
		}
		validVlans := map[AllianceVlans]struct{}{Vlans102030: {}, Vlans405060: {}, Vlans708090: {}}
		if _, ok := validVlans[request.RedVlans]; !ok {
			return fmt.Errorf("invalid value for red VLANs: %s", request.RedVlans)
		}
		if _, ok := validVlans[request.BlueVlans]; !ok {
			return fmt.Errorf("invalid value for blue VLANs: %s", request.BlueVlans)
		}
		if request.RedVlans == request.BlueVlans {
			return fmt.Errorf("red and blue VLANs cannot be the same")
		}
	}

	// Validate station configurations.
	for stationName, stationConfiguration := range request.StationConfigurations {
		stationNameValid := false
		for name := red1; name <= blue3; name++ {
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
		if len(stationConfiguration.WpaKey) < minWpaKeyLength || len(stationConfiguration.WpaKey) > maxWpaKeyLength {
			return fmt.Errorf(
				"invalid WPA key length for station %s: %d (expecting %d-%d)",
				stationName,
				len(stationConfiguration.WpaKey),
				minWpaKeyLength,
				maxWpaKeyLength,
			)
		}
	}

	// Validate syslog IP address.
	if request.SyslogIpAddress != "" {
		match, _ := regexp.MatchString("^((25[0-5]|(2[0-4]|1\\d|[1-9]|)\\d)\\.?\\b){4}$", request.SyslogIpAddress)
		if !match {
			return fmt.Errorf("invalid syslog IP address: %s", request.SyslogIpAddress)
		}
	}

	return nil
}
