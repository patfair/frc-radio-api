// This file is specific to the robot radio version of the API.
//go:build robot

package radio

import (
	"fmt"
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

	// Team-specific WPA key. Must be at least eight characters long.
	WpaKey string `json:"wpaKey"`
}

// Validate checks that all parameters within the configuration request have valid values.
func (request ConfigurationRequest) Validate(radio *Radio) error {
	if request.Mode != modeTeamRadio && request.Mode != modeTeamAccessPoint {
		return fmt.Errorf("invalid operation mode: %s", request.Mode)
	}

	if request.Mode == modeTeamRadio && request.Channel != 0 {
		return fmt.Errorf("channel cannot be set in %s mode", modeTeamRadio)
	}
	if request.Mode == modeTeamAccessPoint && request.Channel != 0 && !isValid6GhzChannel(request.Channel) {
		return fmt.Errorf("invalid 6GHz channel: %d", request.Channel)
	}

	if request.TeamNumber < 1 || request.TeamNumber > 25499 {
		return fmt.Errorf("invalid team number: %d", request.TeamNumber)
	}

	if len(request.WpaKey) < minWpaKeyLength || len(request.WpaKey) > maxWpaKeyLength {
		return fmt.Errorf(
			"invalid WPA key length: %d (expecting %d-%d)", len(request.WpaKey), minWpaKeyLength, maxWpaKeyLength,
		)
	}

	return nil
}
