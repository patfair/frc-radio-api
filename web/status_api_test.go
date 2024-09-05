// This file is specific to the access point version of the API.
//go:build !robot

package web

import (
	"encoding/json"
	"github.com/patfair/frc-radio-api/radio"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWeb_statusHandler(t *testing.T) {
	ap := radio.NewRadio()
	web := NewWebServer(ap)

	ap.Channel = 136
	ap.Status = "ACTIVE"
	ap.StationStatuses["blue1"] = &radio.NetworkStatus{
		Ssid:              "254",
		HashedWpaKey:      "foo",
		WpaKeySalt:        "bar",
		IsLinked:          true,
		RxRateMbps:        1.0,
		TxRateMbps:        2.0,
		SignalNoiseRatio:  3,
		BandwidthUsedMbps: 4.0,
		ConnectionQuality: "warning",
	}

	recorder := web.getHttpResponse("/status")
	assert.Equal(t, 200, recorder.Code)

	var actualAp radio.Radio
	assert.Nil(t, json.Unmarshal(recorder.Body.Bytes(), &actualAp))
	assert.Equal(t, ap.Status, actualAp.Status)
	assert.Equal(t, ap.Status, actualAp.Status)
	assert.Equal(t, ap.StationStatuses, actualAp.StationStatuses)
}

func TestWeb_statusHandlerAuthorization(t *testing.T) {
	ap := radio.NewRadio()
	web := NewWebServer(ap)
	web.password = "mypassword"

	// Without password.
	recorder := web.getHttpResponse("/status")
	assert.Equal(t, 401, recorder.Code)

	// With wrong password.
	recorder = web.getHttpResponseWithHeaders("/status", map[string]string{"Authorization": "Bearer wrongpassword"})
	assert.Equal(t, 401, recorder.Code)

	// With correct password.
	recorder = web.getHttpResponseWithHeaders("/status", map[string]string{"Authorization": "Bearer mypassword"})
	assert.Equal(t, 200, recorder.Code)
}
