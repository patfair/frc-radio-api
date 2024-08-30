// This file is specific to the robot radio version of the API.
//go:build robot

package radio

import (
	"fmt"
	"regexp"
)

const (
	// Maximum length for the SSID suffix.
	maxSsidSuffixLength = 8

	// Regex to validate the SSID suffix.
	ssidSuffixRegex = "^[a-zA-Z0-9]*$"
)

// ConfigurationRequest represents a JSON request to configure the radio.
type ConfigurationRequest struct {
	// Operation mode to configure the radio for.
	Mode radioMode `json:"mode"`

	// 6GHz channel number for the radio to use. If not specified and the radio is configured for TEAM_ACCESS_POINT
	// mode, the radio will automatically select a channel.
	Channel int `json:"channel"`

	// Team number to configure the radio for. Must be between 1 and 25499.
	TeamNumber int `json:"teamNumber"`

	// Suffix to be appended to all WPA SSIDs. Must be alphanumeric and less than eight charaters long.
	SsidSuffix string `json:"ssidSuffix"`

	// Team-specific WPA key for the 6GHz network used by the FMS. Must be at least eight characters long.
	WpaKey6 string `json:"wpaKey6"`

	// WPA key for the 2.4GHz network broadcast by the radio for team use. Must be at least eight characters long.
	WpaKey24 string `json:"wpaKey24"`
}

// Validate checks that all parameters within the configuration request have valid values.
func (request ConfigurationRequest) Validate(radio *Radio) error {
	if request.Mode != modeTeamRobotRadio && request.Mode != modeTeamAccessPoint {
		return fmt.Errorf("invalid operation mode: %s", request.Mode)
	}

	if request.Mode == modeTeamRobotRadio && request.Channel != 0 {
		return fmt.Errorf("channel cannot be set in %s mode", modeTeamRobotRadio)
	}
	if request.Mode == modeTeamAccessPoint && request.Channel != 0 && !isValid6GhzChannel(request.Channel) {
		return fmt.Errorf("invalid 6GHz channel: %d", request.Channel)
	}

	if request.TeamNumber < 1 || request.TeamNumber > 25499 {
		return fmt.Errorf("invalid team number: %d", request.TeamNumber)
	}

	if len(request.SsidSuffix) > maxSsidSuffixLength {
		return fmt.Errorf("invalid ssidSuffix length: %d (expecting 0-%d)", len(request.SsidSuffix), maxSsidSuffixLength)
	}
	if !regexp.MustCompile(ssidSuffixRegex).MatchString(request.SsidSuffix) {
		return fmt.Errorf("invalid ssidSuffix: %s (expecting alphanumeric)", request.SsidSuffix)
	}

	if len(request.WpaKey6) < minWpaKeyLength || len(request.WpaKey6) > maxWpaKeyLength {
		return fmt.Errorf(
			"invalid wpaKey6 length: %d (expecting %d-%d)", len(request.WpaKey6), minWpaKeyLength, maxWpaKeyLength,
		)
	}

	if len(request.WpaKey24) < minWpaKeyLength || len(request.WpaKey24) > maxWpaKeyLength {
		return fmt.Errorf(
			"invalid wpaKey24 length: %d (expecting %d-%d)", len(request.WpaKey24), minWpaKeyLength, maxWpaKeyLength,
		)
	}

	return nil
}
