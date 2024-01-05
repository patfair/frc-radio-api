// This file is specific to the robot radio version of the API.
//go:build robot

package web

import (
	"testing"

	"github.com/patfair/frc-radio-api/radio"
	"github.com/stretchr/testify/assert"
)

func TestWeb_configurationPageHandler(t *testing.T) {
	ap := radio.NewRadio()
	web := NewWebServer(ap)

	// Ensure request results in html returned
	recorder := web.getHttpResponse("/configuration")
	assert.Equal(t, 200, recorder.Code)

	assert.Contains(t, recorder.Body.String(), "</html>")
	assert.Contains(t, recorder.Body.String(), "Configure")
}
