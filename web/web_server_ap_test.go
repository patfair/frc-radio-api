// This file is specific to the access point version of the API.
//go:build !robot

package web

import (
	"github.com/patfair/frc-radio-api/radio"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetVlan100IpAddress(t *testing.T) {
	ipAddress, err := getVlan100IpAddress()
	r := &radio.Radio{Type: radio.TypeLinksys}

	// Branch the test verification logic since it may or may not be Run on a system with a 10.0.100.x interface and
	// mocking the system calls to be deterministic is onerous.
	if err == nil {
		assert.Regexp(t, "^10\\.0\\.100\\.\\d+$", ipAddress)
		assert.Equal(t, ipAddress+":8081", getListenAddress(r))
	} else {
		assert.Contains(t, err.Error(), "no IP address found on VLAN 100")
	}

	// Change the type to Vivid-Hosting and check that the port is different.
	r.Type = radio.TypeVividHosting
	if err == nil {
		assert.Regexp(t, "^10\\.0\\.100\\.\\d+$", ipAddress)
		assert.Equal(t, ipAddress+":80", getListenAddress(r))
	} else {
		assert.Contains(t, err.Error(), "no IP address found on VLAN 100")
	}
}

func TestWeb_rootHandler(t *testing.T) {
	var web WebServer
	recorder := web.getHttpResponse("/")
	assert.Equal(t, 302, recorder.Code)
	assert.Equal(t, "/status", recorder.Header().Get("Location"))
}
