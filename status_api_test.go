package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWeb_statusHandler(t *testing.T) {
	ap := newAccessPoint()
	web := newWeb(ap)

	ap.Channel = 136
	ap.Status = statusActive
	ap.StationStatuses[blue1.String()] = &stationStatus{
		Ssid:               "254",
		HashedWpaKey:       "foo",
		WpaKeySalt:         "bar",
		IsRobotRadioLinked: true,
		RxRateMbps:         1.0,
		TxRateMbps:         2.0,
		SignalNoiseRatio:   3,
		BandwidthUsedMbps:  4.0,
	}

	recorder := web.getHttpResponse("/status")
	assert.Equal(t, 200, recorder.Code)

	var actualAp accessPoint
	assert.Nil(t, json.Unmarshal(recorder.Body.Bytes(), &actualAp))
	assert.Equal(t, ap.Status, actualAp.Status)
	assert.Equal(t, ap.Status, actualAp.Status)
	assert.Equal(t, ap.StationStatuses, actualAp.StationStatuses)
}
