// This file is specific to the robot radio version of the API.
//go:build robot

package radio

import (
	"fmt"
)

// ConfigurationRequest represents a JSON request to configure the radio.
type ConfigurationRequest struct {
	// Team number to configure the radio for. Must be between 1 and 25499.
	TeamNumber int `json:"teamNumber"`

	// Team-specific WPA key. Must be at least eight characters long.
	WpaKey string `json:"wpaKey"`
}

// Validate checks that all parameters within the configuration request have valid values.
func (request ConfigurationRequest) Validate(radio *Radio) error {
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
