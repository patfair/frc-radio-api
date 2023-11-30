// This file is specific to the robot radio version of the API.
//go:build robot

package radio

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestNewRadio(t *testing.T) {
	radio := NewRadio()
	assert.Equal(t, statusBooting, radio.Status)
	assert.NotNil(t, radio.ConfigurationRequestChannel)
}

func TestRadio_isStarted(t *testing.T) {
	fakeShell := newFakeShell(t)
	shell = fakeShell
	radio := NewRadio()

	// Radio is not started.
	fakeShell.commandErrors["iwinfo ath1 info"] = errors.New("failed")
	assert.False(t, radio.isStarted())
	_, ok := fakeShell.commandsRun["iwinfo ath1 info"]
	assert.True(t, ok)

	// Radio is started.
	fakeShell.reset()
	fakeShell.commandOutput["iwinfo ath1 info"] = "some output"
	assert.True(t, radio.isStarted())
	_, ok = fakeShell.commandsRun["iwinfo ath1 info"]
	assert.True(t, ok)
}

func TestRadio_setInitialState(t *testing.T) {
	rand.Seed(0)
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	radio := NewRadio()

	fakeTree.valuesForGet["wireless.@wifi-iface[0].ssid"] = "12345"
	fakeTree.valuesForGet["wireless.@wifi-iface[0].key"] = "11111111"
	radio.setInitialState()
	assert.Equal(t, 12345, radio.TeamNumber)
	assert.Equal(t, "12345", radio.Ssid)
	assert.Equal(t, "c10cc0a95c29b83a73a3d0730f77bbf852016ea4f08aaf5d4291017c6c23bffd", radio.HashedWpaKey)
	assert.Equal(t, "mUNERA9rI2cvTK4U", radio.WpaKeySalt)
}

func TestRadio_handleConfigurationRequest(t *testing.T) {
	rand.Seed(0)
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeShell := newFakeShell(t)
	shell = fakeShell
	radio := NewRadio()

	fakeShell.commandOutput["wifi reload wifi1"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "ath1\nESSID: \"12345\"\n"
	fakeTree.valuesForGet["wireless.@wifi-iface[0].key"] = "11111111"
	dummyRequest1 := ConfigurationRequest{TeamNumber: 1, WpaKey: "foo"}
	dummyRequest2 := ConfigurationRequest{TeamNumber: 2, WpaKey: "bar"}
	request := ConfigurationRequest{TeamNumber: 12345, WpaKey: "11111111"}
	radio.ConfigurationRequestChannel <- dummyRequest2
	radio.ConfigurationRequestChannel <- request
	assert.Nil(t, radio.handleConfigurationRequest(dummyRequest1))
	assert.Equal(t, 7, fakeTree.setCount)
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[0].ssid"], "12345")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[0].key"], "11111111")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.lan.dhcp_option"], "3,10.123.45.4")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.@host[0].name"], "roboRIO-12345-FRC")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.@host[0].ip"], "10.123.45.2")
	assert.Equal(t, fakeTree.valuesFromSet["network.lan.ipaddr"], "10.123.45.1")
	assert.Equal(t, fakeTree.valuesFromSet["network.lan.gateway"], "10.123.45.4")
	assert.Equal(t, 1, fakeTree.commitCount)
	assert.Contains(t, fakeShell.commandsRun, "wifi reload wifi1")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo ath1 info")

	assert.Equal(t, 12345, radio.TeamNumber)
	assert.Equal(t, "12345", radio.Ssid)
	assert.Equal(t, "c10cc0a95c29b83a73a3d0730f77bbf852016ea4f08aaf5d4291017c6c23bffd", radio.HashedWpaKey)
	assert.Equal(t, "mUNERA9rI2cvTK4U", radio.WpaKeySalt)
	assert.Equal(t, radioStatus(statusActive), radio.Status)
}

func TestRadio_handleConfigurationRequestErrors(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeShell := newFakeShell(t)
	shell = fakeShell
	retryBackoffDuration = 10 * time.Millisecond
	radio := NewRadio()

	// wifi reload fails.
	fakeShell.commandErrors["wifi reload wifi1"] = errors.New("oops")
	request := ConfigurationRequest{TeamNumber: 1, WpaKey: "foo"}
	assert.Equal(
		t,
		"failed to reload Wi-Fi configuration for device wifi1: oops",
		radio.handleConfigurationRequest(request).Error(),
	)

	// iwinfo fails.
	fakeTree.reset()
	fakeShell.reset()
	fakeShell.commandOutput["wifi reload wifi1"] = ""
	fakeShell.commandErrors["iwinfo ath1 info"] = errors.New("oops")
	assert.Equal(
		t, "error getting iwinfo for interface ath1: oops", radio.handleConfigurationRequest(request).Error(),
	)

	// iwinfo output is invalid.
	fakeTree.reset()
	fakeShell.reset()
	fakeShell.commandOutput["wifi reload wifi1"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "invalid"
	assert.Equal(
		t, "error parsing iwinfo output for interface ath1: invalid", radio.handleConfigurationRequest(request).Error(),
	)

	// Loop keeps retrying when configuration is incorrect.
	fakeTree.reset()
	fakeShell.reset()
	fakeShell.commandOutput["wifi reload wifi1"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "ath1\nESSID: \"2\"\n"
	go func() {
		time.Sleep(100 * time.Millisecond)
		fakeShell.commandOutput["iwinfo ath1 info"] = "ath1\nESSID: \"1\"\n"
	}()
	assert.Nil(t, radio.handleConfigurationRequest(request))
	assert.Greater(t, fakeTree.commitCount, 5)
}
