package radio

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRadio(t *testing.T) {
	var tree fakeUciTree
	uciTree = &tree

	// Using Vivid-Hosting radio.
	tree.valuesForGet = map[string]string{"system.@system[0].model": "VH-109(AP)"}
	radio := NewRadio()
	assert.Equal(t, 0, radio.Channel)
	assert.Equal(t, statusBooting, radio.Status)
	if assert.Equal(t, int(stationCount), len(radio.StationStatuses)) {
		for i := 0; i < int(stationCount); i++ {
			stationStatus, ok := radio.StationStatuses[station(i).String()]
			assert.True(t, ok)
			assert.Nil(t, stationStatus)
		}
	}
	assert.Equal(t, typeVividHosting, radio.Type)
	assert.NotNil(t, radio.ConfigurationRequestChannel)
	assert.Equal(t, "wifi1", radio.device)
	assert.Equal(
		t,
		map[station]string{
			red1:  "ath1",
			red2:  "ath11",
			red3:  "ath12",
			blue1: "ath13",
			blue2: "ath14",
			blue3: "ath15",
		},
		radio.stationInterfaces,
	)

	// Using Linksys radio.
	tree.valuesForGet["system.@system[0].model"] = ""
	radio = NewRadio()
	assert.Equal(t, 0, radio.Channel)
	assert.Equal(t, statusBooting, radio.Status)
	if assert.Equal(t, int(stationCount), len(radio.StationStatuses)) {
		for i := 0; i < int(stationCount); i++ {
			stationStatus, ok := radio.StationStatuses[station(i).String()]
			assert.True(t, ok)
			assert.Nil(t, stationStatus)
		}
	}
	assert.Equal(t, typeLinksys, radio.Type)
	assert.NotNil(t, radio.ConfigurationRequestChannel)
	assert.Equal(t, "radio0", radio.device)
	assert.Equal(
		t,
		map[station]string{
			red1:  "wlan0",
			red2:  "wlan0-1",
			red3:  "wlan0-2",
			blue1: "wlan0-3",
			blue2: "wlan0-4",
			blue3: "wlan0-5",
		},
		radio.stationInterfaces,
	)
}
