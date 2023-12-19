// This file is specific to the robot radio version of the API.
//go:build robot

package radio

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigurationRequest_Validate(t *testing.T) {
	radio := &Radio{}

	// Invalid team number.
	request := ConfigurationRequest{TeamNumber: 0, WpaKey: "12345678"}
	err := request.Validate(radio)
	assert.EqualError(t, err, "invalid team number: 0")
	request.TeamNumber = 25500
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid team number: 25500")

	// Too-short WPA key.
	request.TeamNumber = 254
	request.WpaKey = "1234567"
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid WPA key length: 7 (expecting 8-16)")

	// Too-long WPA key.
	request.TeamNumber = 254
	request.WpaKey = "12345678123456789"
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid WPA key length: 17 (expecting 8-16)")
}
