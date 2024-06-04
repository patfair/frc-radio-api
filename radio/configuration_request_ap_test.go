// This file is specific to the access point version of the API.
//go:build !robot

package radio

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfigurationRequest_Validate(t *testing.T) {
	linksysRadio := &Radio{Type: TypeLinksys}
	vividHostingRadio := &Radio{Type: TypeVividHosting}

	// Empty request.
	request := ConfigurationRequest{}
	err := request.Validate(linksysRadio)
	assert.EqualError(t, err, "empty configuration request")

	// Invalid 5GHz channel.
	request.Channel = 5
	err = request.Validate(linksysRadio)
	assert.EqualError(t, err, "invalid channel for TypeLinksys: 5")

	// Invalid 6GHz channel.
	request.Channel = 36
	err = request.Validate(vividHostingRadio)
	assert.EqualError(t, err, "invalid channel for TypeVividHosting: 36")

	// Invalid channel bandwidth.
	request = ConfigurationRequest{ChannelBandwidth: "30MHz"}
	err = request.Validate(vividHostingRadio)
	assert.EqualError(t, err, "invalid channel bandwidth: 30MHz")

	// Channel bandwidth not supported on Linksys.
	request = ConfigurationRequest{ChannelBandwidth: "20MHz"}
	err = request.Validate(linksysRadio)
	assert.EqualError(t, err, "channel bandwidth cannot be changed on TypeLinksys")

	// Invalid station.
	request = ConfigurationRequest{
		StationConfigurations: map[string]StationConfiguration{"red4": {Ssid: "254", WpaKey: "12345678"}},
	}
	err = request.Validate(linksysRadio)
	assert.EqualError(t, err, "invalid station: red4")

	// Blank SSID.
	request = ConfigurationRequest{
		StationConfigurations: map[string]StationConfiguration{"blue1": {Ssid: "", WpaKey: "12345678"}},
	}
	err = request.Validate(linksysRadio)
	assert.EqualError(t, err, "SSID for station blue1 cannot be blank")

	// Too-short WPA key.
	request = ConfigurationRequest{
		StationConfigurations: map[string]StationConfiguration{"blue1": {Ssid: "254", WpaKey: "1234567"}},
	}
	err = request.Validate(linksysRadio)
	assert.EqualError(t, err, "invalid WPA key length for station blue1: 7 (expecting 8-16)")

	// Too-long WPA key.
	request = ConfigurationRequest{
		StationConfigurations: map[string]StationConfiguration{"blue1": {Ssid: "254", WpaKey: "12345678123456789"}},
	}
	err = request.Validate(linksysRadio)
	assert.EqualError(t, err, "invalid WPA key length for station blue1: 17 (expecting 8-16)")
}
