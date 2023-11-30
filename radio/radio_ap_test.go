// This file is specific to the access point version of the API.
//go:build !robot

package radio

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewRadio(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree

	// Using Vivid-Hosting radio.
	fakeTree.valuesForGet["system.@system[0].model"] = "VH-109(AP)"
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
	fakeTree.valuesForGet["system.@system[0].model"] = ""
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

func TestRadio_isStarted(t *testing.T) {
	fakeShell := newFakeShell(t)
	shell = fakeShell
	radio := NewRadio()

	// Radio is not started.
	fakeShell.commandErrors["iwinfo wlan0-5 info"] = errors.New("failed")
	assert.False(t, radio.isStarted())
	_, ok := fakeShell.commandsRun["iwinfo wlan0-5 info"]
	assert.True(t, ok)

	// Radio is started.
	fakeShell.reset()
	fakeShell.commandOutput["iwinfo wlan0-5 info"] = "some output"
	assert.True(t, radio.isStarted())
	_, ok = fakeShell.commandsRun["iwinfo wlan0-5 info"]
	assert.True(t, ok)
}

func TestRadio_setInitialState(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeTree.valuesForGet["system.@system[0].model"] = "VH-109(AP)"
	fakeShell := newFakeShell(t)
	shell = fakeShell
	radio := NewRadio()

	fakeTree.valuesForGet["wireless.wifi1.channel"] = "23"
	fakeShell.commandOutput["iwinfo ath1 info"] = "ath1\nESSID: \"1111\"\n"
	fakeShell.commandOutput["iwinfo ath11 info"] = "ath11\nESSID: \"no-team-2\"\n"
	fakeShell.commandOutput["iwinfo ath12 info"] = "ath12\nESSID: \"no-team-3\"\n"
	fakeShell.commandOutput["iwinfo ath13 info"] = "ath13\nESSID: \"no-team-4\"\n"
	fakeShell.commandOutput["iwinfo ath14 info"] = "ath14\nESSID: \"no-team-5\"\n"
	fakeShell.commandOutput["iwinfo ath15 info"] = "ath15\nESSID: \"6666\"\n"
	radio.setInitialState()
	assert.Equal(t, 23, radio.Channel)
	assert.Equal(t, "1111", radio.StationStatuses["red1"].Ssid)
	assert.Nil(t, radio.StationStatuses["red2"])
	assert.Nil(t, radio.StationStatuses["red3"])
	assert.Nil(t, radio.StationStatuses["blue1"])
	assert.Nil(t, radio.StationStatuses["blue2"])
	assert.Equal(t, "6666", radio.StationStatuses["blue3"].Ssid)
}

func TestRadio_handleConfigurationRequestVividHosting(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeTree.valuesForGet["system.@system[0].model"] = "VH-109(AP)"
	fakeShell := newFakeShell(t)
	shell = fakeShell
	radio := NewRadio()

	fakeShell.commandOutput["wifi reload wifi1"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "ath1\nESSID: \"1111\"\n"
	fakeShell.commandOutput["iwinfo ath11 info"] = "ath11\nESSID: \"no-team-2\"\n"
	fakeShell.commandOutput["iwinfo ath12 info"] = "ath12\nESSID: \"3333\"\n"
	fakeShell.commandOutput["iwinfo ath13 info"] = "ath13\nESSID: \"no-team-4\"\n"
	fakeShell.commandOutput["iwinfo ath14 info"] = "ath14\nESSID: \"5555\"\n"
	fakeShell.commandOutput["iwinfo ath15 info"] = "ath15\nESSID: \"6666\"\n"
	dummyRequest1 := ConfigurationRequest{
		Channel:               1,
		StationConfigurations: map[string]StationConfiguration{"red1": {Ssid: "1", WpaKey: "foo"}},
	}
	dummyRequest2 := ConfigurationRequest{
		Channel:               2,
		StationConfigurations: map[string]StationConfiguration{"blue2": {Ssid: "2", WpaKey: "bar"}},
	}
	request := ConfigurationRequest{
		Channel: 5,
		StationConfigurations: map[string]StationConfiguration{
			"red1":  {Ssid: "1111", WpaKey: "11111111"},
			"red3":  {Ssid: "3333", WpaKey: "33333333"},
			"blue2": {Ssid: "5555", WpaKey: "55555555"},
			"blue3": {Ssid: "6666", WpaKey: "66666666"},
		},
	}
	radio.ConfigurationRequestChannel <- dummyRequest2
	radio.ConfigurationRequestChannel <- request
	assert.Nil(t, radio.handleConfigurationRequest(dummyRequest1))
	assert.Equal(t, 19, fakeTree.setCount)
	assert.Equal(t, fakeTree.valuesFromSet["wireless.wifi1.channel"], "5")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].ssid"], "1111")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].key"], "11111111")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].sae_password"], "11111111")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[2].ssid"], "no-team-2")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[2].key"], "no-team-2")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[2].sae_password"], "no-team-2")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[3].ssid"], "3333")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[3].key"], "33333333")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[3].sae_password"], "33333333")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[4].ssid"], "no-team-4")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[4].key"], "no-team-4")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[4].sae_password"], "no-team-4")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[5].ssid"], "5555")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[5].key"], "55555555")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[5].sae_password"], "55555555")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[6].ssid"], "6666")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[6].key"], "66666666")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[6].sae_password"], "66666666")
	assert.Equal(t, 6, fakeTree.commitCount)
	assert.Contains(t, fakeShell.commandsRun, "wifi reload wifi1")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo ath1 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo ath11 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo ath12 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo ath13 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo ath14 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo ath15 info")

	assert.Equal(t, "1111", radio.StationStatuses["red1"].Ssid)
	assert.Nil(t, radio.StationStatuses["red2"])
	assert.Equal(t, "3333", radio.StationStatuses["red3"].Ssid)
	assert.Nil(t, radio.StationStatuses["blue1"])
	assert.Equal(t, "5555", radio.StationStatuses["blue2"].Ssid)
	assert.Equal(t, "6666", radio.StationStatuses["blue3"].Ssid)
}

func TestRadio_handleConfigurationRequestLinksys(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeTree.valuesForGet["system.@system[0].model"] = ""
	fakeShell := newFakeShell(t)
	shell = fakeShell
	linksysWifiReloadBackoffDuration = 100 * time.Millisecond
	radio := NewRadio()

	fakeShell.commandOutput["wifi reload radio0"] = ""
	fakeShell.commandOutput["iwinfo wlan0 info"] = "wlan0\nESSID: \"no-team-1\"\n"
	fakeShell.commandOutput["iwinfo wlan0-1 info"] = "wlan0-1\nESSID: \"no-team-2\"\n"
	fakeShell.commandOutput["iwinfo wlan0-2 info"] = "wlan0-2\nESSID: \"no-team-3\"\n"
	fakeShell.commandOutput["iwinfo wlan0-3 info"] = "wlan0-3\nESSID: \"no-team-4\"\n"
	fakeShell.commandOutput["iwinfo wlan0-4 info"] = "wlan0-4\nESSID: \"no-team-5\"\n"
	fakeShell.commandOutput["iwinfo wlan0-5 info"] = "wlan0-5\nESSID: \"no-team-6\"\n"
	dummyRequest1 := ConfigurationRequest{
		Channel:               1,
		StationConfigurations: map[string]StationConfiguration{"red1": {Ssid: "1", WpaKey: "foo"}},
	}
	dummyRequest2 := ConfigurationRequest{
		Channel:               2,
		StationConfigurations: map[string]StationConfiguration{"blue2": {Ssid: "2", WpaKey: "bar"}},
	}
	request := ConfigurationRequest{
		Channel: 5,
		StationConfigurations: map[string]StationConfiguration{
			"red2":  {Ssid: "2222", WpaKey: "22222222"},
			"red3":  {Ssid: "3333", WpaKey: "33333333"},
			"blue1": {Ssid: "4444", WpaKey: "44444444"},
			"blue2": {Ssid: "5555", WpaKey: "55555555"},
		},
	}
	radio.ConfigurationRequestChannel <- dummyRequest2
	radio.ConfigurationRequestChannel <- request
	go func() {
		// Allow some time for the first config-clearing change to be processed.
		time.Sleep(150 * time.Millisecond)

		assert.Equal(t, 13, fakeTree.setCount)
		assert.Equal(t, fakeTree.valuesFromSet["wireless.radio0.channel"], "5")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].ssid"], "no-team-1")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].key"], "no-team-1")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[2].ssid"], "no-team-2")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[2].key"], "no-team-2")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[3].ssid"], "no-team-3")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[3].key"], "no-team-3")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[4].ssid"], "no-team-4")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[4].key"], "no-team-4")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[5].ssid"], "no-team-5")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[5].key"], "no-team-5")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[6].ssid"], "no-team-6")
		assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[6].key"], "no-team-6")
		assert.Equal(t, 6, fakeTree.commitCount)
		assert.Contains(t, fakeShell.commandsRun, "wifi reload radio0")
		assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0 info")
		assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-1 info")
		assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-2 info")
		assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-3 info")
		assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-4 info")
		assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-5 info")
		fakeTree.reset()
		fakeShell.reset()

		// Change the iwinfo output after the configs are cleared.
		fakeShell.commandOutput["wifi reload radio0"] = ""
		fakeShell.commandOutput["iwinfo wlan0 info"] = "wlan0\nESSID: \"no-team-1\"\n"
		fakeShell.commandOutput["iwinfo wlan0-1 info"] = "wlan0-1\nESSID: \"2222\"\n"
		fakeShell.commandOutput["iwinfo wlan0-2 info"] = "wlan0-2\nESSID: \"3333\"\n"
		fakeShell.commandOutput["iwinfo wlan0-3 info"] = "wlan0-3\nESSID: \"4444\"\n"
		fakeShell.commandOutput["iwinfo wlan0-4 info"] = "wlan0-4\nESSID: \"5555\"\n"
		fakeShell.commandOutput["iwinfo wlan0-5 info"] = "wlan0-5\nESSID: \"no-team-6\"\n"
	}()
	assert.Nil(t, radio.handleConfigurationRequest(dummyRequest1))
	assert.Equal(t, 12, fakeTree.setCount)
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].ssid"], "no-team-1")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[1].key"], "no-team-1")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[2].ssid"], "2222")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[2].key"], "22222222")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[3].ssid"], "3333")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[3].key"], "33333333")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[4].ssid"], "4444")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[4].key"], "44444444")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[5].ssid"], "5555")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[5].key"], "55555555")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[6].ssid"], "no-team-6")
	assert.Equal(t, fakeTree.valuesFromSet["wireless.@wifi-iface[6].key"], "no-team-6")
	assert.Equal(t, 6, fakeTree.commitCount)
	assert.Contains(t, fakeShell.commandsRun, "wifi reload radio0")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-1 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-2 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-3 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-4 info")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-5 info")
}

func TestRadio_handleConfigurationRequestErrors(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeTree.valuesForGet["system.@system[0].model"] = "VH-109(AP)"
	fakeShell := newFakeShell(t)
	shell = fakeShell
	retryBackoffDuration = 1 * time.Millisecond
	radio := NewRadio()

	// wifi reload fails.
	fakeShell.commandErrors["wifi reload wifi1"] = errors.New("oops")
	request := ConfigurationRequest{Channel: 5}
	assert.Equal(
		t, "failed to reload configuration for device wifi1: oops", radio.handleConfigurationRequest(request).Error(),
	)

	// iwinfo fails.
	fakeTree.reset()
	fakeShell.reset()
	fakeShell.commandOutput["wifi reload wifi1"] = ""
	fakeShell.commandErrors["iwinfo ath1 info"] = errors.New("oops")
	fakeShell.commandOutput["iwinfo ath11 info"] = "ath11\nESSID: \"no-team-2\"\n"
	fakeShell.commandOutput["iwinfo ath12 info"] = "ath12\nESSID: \"no-team-3\"\n"
	fakeShell.commandOutput["iwinfo ath13 info"] = "ath13\nESSID: \"no-team-4\"\n"
	fakeShell.commandOutput["iwinfo ath14 info"] = "ath14\nESSID: \"no-team-5\"\n"
	fakeShell.commandOutput["iwinfo ath15 info"] = "ath15\nESSID: \"no-team-6\"\n"
	assert.Equal(
		t,
		"error updating station statuses: error getting iwinfo for interface ath1: oops",
		radio.handleConfigurationRequest(request).Error(),
	)

	// iwinfo output is invalid.
	fakeTree.reset()
	fakeShell.reset()
	fakeShell.commandOutput["wifi reload wifi1"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "ath1\nESSID: \"1111\"\n"
	fakeShell.commandOutput["iwinfo ath11 info"] = "ath11\nESSID: \"no-team-2\"\n"
	fakeShell.commandOutput["iwinfo ath12 info"] = "ath12\nESSID: \"3333\"\n"
	fakeShell.commandOutput["iwinfo ath13 info"] = "ath13\nESSID: \"no-team-4\"\n"
	fakeShell.commandOutput["iwinfo ath14 info"] = "invalid"
	fakeShell.commandOutput["iwinfo ath15 info"] = "ath15\nESSID: \"6666\"\n"
	assert.Equal(
		t,
		"error updating station statuses: error parsing iwinfo output for interface ath14: invalid",
		radio.handleConfigurationRequest(request).Error(),
	)

	// Loop keeps retrying when configuration is incorrect.
	fakeTree.reset()
	fakeShell.reset()
	fakeShell.commandOutput["wifi reload wifi1"] = ""
	fakeShell.commandOutput["iwinfo ath1 info"] = "ath1\nESSID: \"1111\"\n"
	fakeShell.commandOutput["iwinfo ath11 info"] = "ath11\nESSID: \"no-team-2\"\n"
	fakeShell.commandOutput["iwinfo ath12 info"] = "ath12\nESSID: \"no-team-3\"\n"
	fakeShell.commandOutput["iwinfo ath13 info"] = "ath13\nESSID: \"no-team-4\"\n"
	fakeShell.commandOutput["iwinfo ath14 info"] = "ath14\nESSID: \"no-team-5\"\n"
	fakeShell.commandOutput["iwinfo ath15 info"] = "ath15\nESSID: \"no-team-6\"\n"
	go func() {
		time.Sleep(10 * time.Millisecond)
		fakeShell.commandOutput["iwinfo ath1 info"] = "ath1\nESSID: \"no-team-1\"\n"
	}()
	assert.Nil(t, radio.handleConfigurationRequest(request))
	assert.Greater(t, fakeTree.commitCount, 20)
}

func TestRadio_updateStationMonitoring(t *testing.T) {
	fakeShell := newFakeShell(t)
	shell = fakeShell
	radio := NewRadio()

	// No teams assigned.
	radio.updateMonitoring()
	assert.Empty(t, fakeShell.commandsRun)

	// Some teams assigned.
	fakeShell.reset()
	radio.StationStatuses["red1"] = &StationStatus{}
	radio.StationStatuses["red3"] = &StationStatus{}
	radio.StationStatuses["blue2"] = &StationStatus{}
	fakeShell.commandErrors["luci-bwc -i wlan0"] = errors.New("oops")
	fakeShell.commandOutput["iwinfo wlan0 assoclist"] = "48:DA:35:B0:00:CF  -53 dBm / -95 dBm (SNR 42)  0 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 254.0 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	fakeShell.commandOutput["luci-bwc -i wlan0-2"] = "[ 1687496917, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496919, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496921, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496922, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496923, 2609700, 177, 7064600, 849 ]"
	fakeShell.commandOutput["iwinfo wlan0-2 assoclist"] = ""
	fakeShell.commandOutput["luci-bwc -i wlan0-4"] = ""
	fakeShell.commandErrors["iwinfo wlan0-4 assoclist"] = errors.New("oops")
	radio.updateMonitoring()
	assert.Equal(
		t,
		StationStatus{
			IsRobotRadioLinked: true,
			RxRateMbps:         550.6,
			TxRateMbps:         254.0,
			SignalNoiseRatio:   42,
			BandwidthUsedMbps:  -999,
		},
		*radio.StationStatuses["red1"],
	)
	assert.Equal(
		t,
		StationStatus{
			IsRobotRadioLinked: false,
			RxRateMbps:         0,
			TxRateMbps:         0,
			SignalNoiseRatio:   0,
			BandwidthUsedMbps:  15.324,
		},
		*radio.StationStatuses["red3"],
	)
	assert.Equal(
		t,
		StationStatus{
			IsRobotRadioLinked: false,
			RxRateMbps:         -999,
			TxRateMbps:         -999,
			SignalNoiseRatio:   -999,
			BandwidthUsedMbps:  0,
		},
		*radio.StationStatuses["blue2"],
	)
	assert.Equal(t, 6, len(fakeShell.commandsRun))
	assert.Contains(t, fakeShell.commandsRun, "luci-bwc -i wlan0")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0 assoclist")
	assert.Contains(t, fakeShell.commandsRun, "luci-bwc -i wlan0-2")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-2 assoclist")
	assert.Contains(t, fakeShell.commandsRun, "luci-bwc -i wlan0-4")
	assert.Contains(t, fakeShell.commandsRun, "iwinfo wlan0-4 assoclist")
}
