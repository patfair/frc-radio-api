package radio

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRadio(t *testing.T) {
	radio := NewRadio()
	assert.Equal(t, statusBooting, radio.Status)
	if assert.Equal(t, int(stationCount), len(radio.StationStatuses)) {
		for i := 0; i < int(stationCount); i++ {
			stationStatus, ok := radio.StationStatuses[station(i).String()]
			assert.True(t, ok)
			assert.Nil(t, stationStatus)
		}
	}
}
