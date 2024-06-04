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
	fakeShell.commandOutput["sh -c source /etc/openwrt_release && echo $DISTRIB_DESCRIPTION"] = ""
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
	fakeShell := newFakeShell(t)
	shell = fakeShell
	fakeShell.commandOutput["sh -c source /etc/openwrt_release && echo $DISTRIB_DESCRIPTION"] = ""
	radio := NewRadio()

	fakeTree.valuesForGet["wireless.@wifi-iface[1].ssid"] = "12345"
	fakeTree.valuesForGet["wireless.@wifi-iface[1].key"] = "11111111"
	radio.setInitialState()
	assert.Equal(t, 12345, radio.TeamNumber)
	assert.Equal(t, "12345", radio.Ssid)
	assert.Equal(t, "c10cc0a95c29b83a73a3d0730f77bbf852016ea4f08aaf5d4291017c6c23bffd", radio.HashedWpaKey)
	assert.Equal(t, "mUNERA9rI2cvTK4U", radio.WpaKeySalt)

	// Test with team radio mode.
	fakeTree.valuesForGet["wireless.@wifi-iface[1].mode"] = "sta"
	fakeTree.valuesForGet["wireless.wifi0.channel"] = "1"
	radio.setInitialState()
	assert.Equal(t, modeTeamRobotRadio, radio.Mode)
	assert.Equal(t, "", radio.Channel)

	// Test with team access point mode.
	fakeTree.valuesForGet["wireless.@wifi-iface[1].mode"] = "ap"
	fakeTree.valuesForGet["wireless.wifi1.channel"] = "36"
	radio.setInitialState()
	assert.Equal(t, modeTeamAccessPoint, radio.Mode)
	assert.Equal(t, "36", radio.Channel)

	// Test with team access point mode and automatic channel.
	fakeTree.valuesForGet["wireless.wifi1.channel"] = "auto"
	radio.setInitialState()
	assert.Equal(t, modeTeamAccessPoint, radio.Mode)
	assert.Equal(t, "auto", radio.Channel)
}

func TestRadio_handleConfigurationRequest(t *testing.T) {
	rand.Seed(0)
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeShell := newFakeShell(t)
	shell = fakeShell
	wifiReloadBackoffDuration = 10 * time.Millisecond
	fakeShell.commandOutput["sh -c source /etc/openwrt_release && echo $DISTRIB_DESCRIPTION"] = ""
	radio := NewRadio()

	// Configure to team radio mode.
	fakeShell.commandOutput["wifi reload"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "ath0\nESSID: \"12345\"\n"
	fakeTree.valuesForGet["wireless.@wifi-iface[1].key"] = "11111111"
	dummyRequest1 := ConfigurationRequest{TeamNumber: 1, WpaKey6: "foo"}
	dummyRequest2 := ConfigurationRequest{TeamNumber: 2, WpaKey6: "bar"}
	request := ConfigurationRequest{
		Mode: modeTeamRobotRadio, TeamNumber: 12345, WpaKey6: "11111111", WpaKey24: "22222222",
	}
	radio.ConfigurationRequestChannel <- dummyRequest2
	radio.ConfigurationRequestChannel <- request
	assert.Nil(t, radio.handleConfigurationRequest(dummyRequest1))
	assert.Equal(t, 18, fakeTree.setCount)
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].ssid"], "12345")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].key"], "11111111")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].mode"], "sta")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[0].ssid"], "FRC-12345")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[0].key"], "22222222")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[0].mode"], "ap")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.wifi1.channel"], "***DELETED***")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.wifi0.channel"], "auto")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.wifi0.disabled"], "0")
	assert.Equal(t, fakeTree.valuesFromSet["network.lan.ipaddr"], "10.123.45.1")
	assert.Equal(t, fakeTree.valuesFromSet["network.lan.gateway"], "10.123.45.4")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.lan.start"], "200")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.lan.limit"], "20")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.@host[0]"], "***ADDED***")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.lan.dhcp_option"], "3,10.123.45.4")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.@host[0].name"], "roboRIO-12345-FRC")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.@host[0].ip"], "10.123.45.2")
	assert.Equal(t, 1, fakeTree.commitCount)
	assert.Contains(t, fakeShell.commandsRun, "wifi reload")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo ath1 info")
	assert.Equal(t, 12345, radio.TeamNumber)
	assert.Equal(t, "12345", radio.Ssid)
	assert.Equal(t, "c10cc0a95c29b83a73a3d0730f77bbf852016ea4f08aaf5d4291017c6c23bffd", radio.HashedWpaKey)
	assert.Equal(t, "mUNERA9rI2cvTK4U", radio.WpaKeySalt)
	assert.Equal(t, statusActive, radio.Status)
	assert.Equal(t, modeTeamRobotRadio, radio.Mode)
	assert.Equal(t, "", radio.Channel)

	// Configure to team access point mode with specified channel.
	fakeShell.reset()
	fakeTree.reset()
	fakeShell.commandOutput["wifi reload"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "ath0\nESSID: \"12345\"\n"
	fakeTree.valuesForGet["wireless.@wifi-iface[1].key"] = "11111111"
	request = ConfigurationRequest{Mode: modeTeamAccessPoint, TeamNumber: 12345, WpaKey6: "11111111", Channel: 229}
	assert.Nil(t, radio.handleConfigurationRequest(request))
	assert.Equal(t, 14, fakeTree.setCount)
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].ssid"], "12345")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].key"], "11111111")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].mode"], "ap")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.wifi1.channel"], "229")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.wifi0.disabled"], "1")
	assert.Equal(t, fakeTree.valuesFromSet["network.lan.ipaddr"], "10.123.45.4")
	assert.Equal(t, fakeTree.valuesFromSet["network.lan.gateway"], "10.123.45.4")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.lan.start"], "20")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.lan.limit"], "180")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.@host[-1]"], "***DELETED***")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.@host[0]"], "***ADDED***")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.lan.dhcp_option"], "3,10.123.45.4")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.@host[0].name"], "roboRIO-12345-FRC")
	assert.Equal(t, fakeTree.valuesFromSet["dhcp.@host[0].ip"], "10.123.45.2")
	assert.Equal(t, 1, fakeTree.commitCount)
	assert.Contains(t, fakeShell.commandsRun, "wifi reload")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo ath1 info")
	assert.Equal(t, 12345, radio.TeamNumber)
	assert.Equal(t, "12345", radio.Ssid)
	assert.Equal(t, "8441e86a503c6028f7d308d18f0eb15e734862db94ce55e9e590c1febdee991c", radio.HashedWpaKey)
	assert.Equal(t, "HomcjcEQvymkzADm", radio.WpaKeySalt)
	assert.Equal(t, statusActive, radio.Status)
	assert.Equal(t, modeTeamAccessPoint, radio.Mode)
	assert.Equal(t, "229", radio.Channel)

	// Configure to team access point mode with automatic channel.
	fakeTree.reset()
	fakeTree.valuesForGet["wireless.@wifi-iface[0].key"] = "11111111"
	request = ConfigurationRequest{Mode: modeTeamAccessPoint, TeamNumber: 12345, WpaKey6: "11111111"}
	assert.Nil(t, radio.handleConfigurationRequest(request))
	assert.Equal(t, fakeTree.valuesFromSet["wireless.wifi1.channel"], "auto")
	assert.Equal(t, modeTeamAccessPoint, radio.Mode)
	assert.Equal(t, "auto", radio.Channel)

	// Configure back to radio mode to ensure status is updated.
	fakeTree.reset()
	fakeTree.valuesForGet["wireless.@wifi-iface[0].key"] = "11111111"
	request = ConfigurationRequest{Mode: modeTeamRobotRadio, TeamNumber: 12345, WpaKey6: "11111111"}
	assert.Nil(t, radio.handleConfigurationRequest(request))
	assert.Equal(t, fakeTree.valuesFromSet["wireless.wifi1.channel"], "***DELETED***")
	assert.Equal(t, modeTeamRobotRadio, radio.Mode)
	assert.Equal(t, "", radio.Channel)
}

func TestRadio_handleConfigurationRequestErrors(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeShell := newFakeShell(t)
	shell = fakeShell
	retryBackoffDuration = 10 * time.Millisecond
	wifiReloadBackoffDuration = 10 * time.Millisecond
	fakeShell.commandOutput["sh -c source /etc/openwrt_release && echo $DISTRIB_DESCRIPTION"] = ""
	radio := NewRadio()

	// wifi reload fails.
	fakeShell.commandErrors["wifi reload"] = errors.New("oops")
	request := ConfigurationRequest{TeamNumber: 1, WpaKey6: "foo"}
	assert.Equal(t, "failed to reload Wi-Fi configuration: oops", radio.handleConfigurationRequest(request).Error())

	// iwinfo fails.
	fakeTree.reset()
	fakeShell.reset()
	fakeShell.commandOutput["wifi reload"] = ""
	fakeShell.commandErrors["iwinfo ath1 info"] = errors.New("oops")
	assert.Equal(t, "error getting iwinfo for interface ath1: oops", radio.handleConfigurationRequest(request).Error())

	// iwinfo output is invalid.
	fakeTree.reset()
	fakeShell.reset()
	fakeShell.commandOutput["wifi reload"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "invalid"
	assert.Equal(
		t, "error parsing iwinfo output for interface ath1: invalid", radio.handleConfigurationRequest(request).Error(),
	)

	// Loop keeps retrying when configuration is incorrect.
	fakeTree.reset()
	fakeShell.reset()
	fakeShell.commandOutput["wifi reload"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "ath0\nESSID: \"2\"\n"
	go func() {
		time.Sleep(100 * time.Millisecond)
		fakeShell.commandOutput["iwinfo ath1 info"] = "ath0\nESSID: \"1\"\n"
	}()
	assert.Nil(t, radio.handleConfigurationRequest(request))
	assert.Greater(t, fakeTree.commitCount, 5)
}
