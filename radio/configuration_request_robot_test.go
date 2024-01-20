// This file is specific to the robot radio version of the API.
//go:build robot

package radio

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigurationRequest_Validate(t *testing.T) {
	radio := &Radio{}

	// Invalid operation mode.
	request := ConfigurationRequest{TeamNumber: 254, WpaKey6: "12345678", WpaKey24: "87654321"}
	request.Mode = "NONEXISTENT_MODE"
	err := request.Validate(radio)
	assert.EqualError(t, err, "invalid operation mode: NONEXISTENT_MODE")

	// Setting channel not allowed in TEAM_RADIO mode.
	request.Mode = modeTeamRobotRadio
	request.Channel = 21
	err = request.Validate(radio)
	assert.EqualError(t, err, "channel cannot be set in TEAM_ROBOT_RADIO mode")

	// Invalid channel.
	request.Mode = modeTeamAccessPoint
	assert.Nil(t, request.Validate(radio))
	request.Channel = 36
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid 6GHz channel: 36")
	request.Channel = 0
	assert.Nil(t, request.Validate(radio))
	request.Mode = modeTeamRobotRadio
	request.Channel = 0

	// Invalid team number.
	request.TeamNumber = 0
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid team number: 0")
	request.TeamNumber = 25500
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid team number: 25500")

	// Too-short 6GHz WPA key.
	request.TeamNumber = 254
	request.WpaKey6 = "1234567"
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid wpaKey6 length: 7 (expecting 8-16)")

	// Too-long 6GHz WPA key.
	request.WpaKey6 = "12345678123456789"
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid wpaKey6 length: 17 (expecting 8-16)")

	// Invalid 6GHz WPA key.
	request.WpaKey6 = "abc123!@#"
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid wpaKey6 (expecting alphanumeric)")

	// Too-short 2.4GHz WPA key.
	request.WpaKey6 = "12345678"
	request.WpaKey24 = "1234567"
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid wpaKey24 length: 7 (expecting 8-16)")

	// Too-long 2.4GHz WPA key.
	request.WpaKey24 = "12345678123456789"
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid wpaKey24 length: 17 (expecting 8-16)")

	// Invalid 2.4GHz WPA key.
	request.WpaKey24 = "abc123!@#"
	err = request.Validate(radio)
	assert.EqualError(t, err, "invalid wpaKey24 (expecting alphanumeric)")
}
