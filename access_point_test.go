package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewAccessPoint(t *testing.T) {
	accessPoint := newAccessPoint()
	assert.Equal(t, statusBooting, accessPoint.Status)
	if assert.Equal(t, int(stationCount), len(accessPoint.StationStatuses)) {
		for i := 0; i < int(stationCount); i++ {
			stationStatus, ok := accessPoint.StationStatuses[station(i).String()]
			assert.True(t, ok)
			assert.Nil(t, stationStatus)
		}
	}
}
