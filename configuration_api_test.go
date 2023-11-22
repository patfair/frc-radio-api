package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWeb_configurationHandler(t *testing.T) {
	ap := newAccessPoint()
	web := newWeb(ap)

	// Empty request should result in an error.
	recorder := web.postHttpResponse("/configuration", "{}")
	assert.Equal(t, 400, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "empty configuration request")
	assert.Equal(t, 0, len(ap.configurationRequestChannel))

	// Request to configure a single team.
	recorder = web.postHttpResponse(
		"/configuration", `{"stationConfigurations": {"blue1": {"ssid": "254", "wpaKey": "foo"}}}`,
	)
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "configuration received")
	if assert.Equal(t, 1, len(ap.configurationRequestChannel)) {
		request := <-ap.configurationRequestChannel
		assert.Equal(t, 0, request.Channel)
		assert.Equal(t, 1, len(request.StationConfigurations))
		assert.Equal(t, stationConfiguration{Ssid: "254", WpaKey: "foo"}, request.StationConfigurations["blue1"])
	}

	// Request to configure everything.
	recorder = web.postHttpResponse(
		"/configuration",
		`
		{
			"channel": 11,
			"stationConfigurations": {
				"red1": {"ssid": "9991", "wpaKey": "11111111"},
				"red2": {"ssid": "9992", "wpaKey": "22222222"},
				"red3": {"ssid": "9993", "wpaKey": "33333333"},
				"blue1": {"ssid": "9994", "wpaKey": "44444444"},
				"blue2": {"ssid": "9995", "wpaKey": "55555555"},
				"blue3": {"ssid": "9996", "wpaKey": "66666666"}
			}
		}
		`,
	)
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "configuration received")
	if assert.Equal(t, 1, len(ap.configurationRequestChannel)) {
		request := <-ap.configurationRequestChannel
		assert.Equal(t, 11, request.Channel)
		assert.Equal(t, 6, len(request.StationConfigurations))
		assert.Equal(t, stationConfiguration{Ssid: "9991", WpaKey: "11111111"}, request.StationConfigurations["red1"])
		assert.Equal(t, stationConfiguration{Ssid: "9992", WpaKey: "22222222"}, request.StationConfigurations["red2"])
		assert.Equal(t, stationConfiguration{Ssid: "9993", WpaKey: "33333333"}, request.StationConfigurations["red3"])
		assert.Equal(t, stationConfiguration{Ssid: "9994", WpaKey: "44444444"}, request.StationConfigurations["blue1"])
		assert.Equal(t, stationConfiguration{Ssid: "9995", WpaKey: "55555555"}, request.StationConfigurations["blue2"])
		assert.Equal(t, stationConfiguration{Ssid: "9996", WpaKey: "66666666"}, request.StationConfigurations["blue3"])
	}
}
