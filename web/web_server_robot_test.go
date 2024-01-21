// This file is specific to the robot radio version of the API.
//go:build robot

package web

import (
	"github.com/patfair/frc-radio-api/radio"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetListenAddress(t *testing.T) {
	r := &radio.Radio{}
	assert.Equal(t, ":80", getListenAddress(r))
}

func TestWeb_rootHandler(t *testing.T) {
	var web WebServer
	recorder := web.getHttpResponse("/")
	assert.Equal(t, 302, recorder.Code)
	assert.Equal(t, "/configuration", recorder.Header().Get("Location"))
}
