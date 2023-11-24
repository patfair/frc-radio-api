package radio

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigurationRequest_Validate(t *testing.T) {
	// Empty request.
	request := ConfigurationRequest{}
	err := request.Validate(typeLinksys)
	assert.EqualError(t, err, "empty configuration request")

	// Invalid 5GHz channel.
	request.Channel = 5
	err = request.Validate(typeLinksys)
	assert.EqualError(t, err, "invalid channel for typeLinksys: 5")

	// Invalid 6GHz channel.
	request.Channel = 36
	err = request.Validate(typeVividHosting)
	assert.EqualError(t, err, "invalid channel for typeVividHosting: 36")

	// Invalid station.
	request = ConfigurationRequest{
		StationConfigurations: map[string]StationConfiguration{"red4": {Ssid: "254", WpaKey: "12345678"}},
	}
	err = request.Validate(typeLinksys)
	assert.EqualError(t, err, "invalid station: red4")

	// Blank SSID.
	request = ConfigurationRequest{
		StationConfigurations: map[string]StationConfiguration{"blue1": {Ssid: "", WpaKey: "12345678"}},
	}
	err = request.Validate(typeLinksys)
	assert.EqualError(t, err, "SSID for station blue1 cannot be blank")

	// Too-short WPA key.
	request = ConfigurationRequest{
		StationConfigurations: map[string]StationConfiguration{"blue1": {Ssid: "254", WpaKey: "1234567"}},
	}
	err = request.Validate(typeLinksys)
	assert.EqualError(t, err, "invalid WPA key length for station blue1: 7")
}
