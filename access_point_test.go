package main

import (
	"github.com/stretchr/testify/assert"
	"math"
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

func TestParseBandwithUsed(t *testing.T) {
	// Response is too short.
	assert.Equal(t, 0.0, parseBandwidthUsed(""))
	response := "[ 1687496957, 26097, 177, 71670, 865 ],\n" +
		"[ 1687496958, 26097, 177, 71734, 866 ],\n" +
		"[ 1687496959, 26097, 177, 71734, 866 ],\n" +
		"[ 1687496960, 26097, 177, 71798, 867 ],\n" +
		"[ 1687496960, 26097, 177, 71798, 867 ],\n" +
		"[ 1687496961, 26097, 177, 71798, 867 ]"
	assert.Equal(t, 0.0, parseBandwidthUsed(response))

	// Response is normal.
	response = "[ 1687496917, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496919, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496921, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496922, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496923, 2609700, 177, 7064600, 849 ]"
	assert.Equal(t, 15.0, math.Floor(parseBandwidthUsed(response)))

	// Response also includes associated client information.
	response = "[ 1687496917, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496919, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496921, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496922, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496923, 2609700, 177, 7064600, 849 ]\n" +
		"48:DA:35:B0:00:CF  -52 dBm / -95 dBm (SNR 43)  1000 ms ago\n" +
		"\tRX: 619.4 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	assert.Equal(t, 15.0, math.Floor(parseBandwidthUsed(response)))
}

func TestStationStatus_ParseAssocList(t *testing.T) {
	var status stationStatus

	status.parseAssocList("")
	assert.Equal(t, stationStatus{}, status)

	// MAC address is invalid.
	response := "00:00:00:00:00:00  -53 dBm / -95 dBm (SNR 42)  0 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(t, stationStatus{}, status)

	// Link is valid.
	response = "48:DA:35:B0:00:CF  -53 dBm / -95 dBm (SNR 42)  0 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 254.0 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(
		t, stationStatus{IsRobotRadioLinked: true, RxRateMbps: 550.6, TxRateMbps: 254.0, SignalNoiseRatio: 42}, status,
	)
	response = "48:DA:35:B0:00:CF  -53 dBm / -95 dBm (SNR 7)  4000 ms ago\n" +
		"\tRX: 123.4 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(
		t, stationStatus{IsRobotRadioLinked: true, RxRateMbps: 123.4, TxRateMbps: 550.6, SignalNoiseRatio: 7}, status,
	)

	// Link is stale.
	response = "48:DA:35:B0:00:CF  -53 dBm / -95 dBm (SNR 42)  4001 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(t, stationStatus{}, status)

	// Response also includes BTU information.
	response = "[ 1687496917, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496919, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496921, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496922, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496923, 2609700, 177, 7064600, 849 ]\n" +
		"48:DA:35:B0:00:CF  -52 dBm / -95 dBm (SNR 43)  1000 ms ago\n" +
		"\tRX: 619.4 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(
		t, stationStatus{IsRobotRadioLinked: true, RxRateMbps: 619.4, TxRateMbps: 550.6, SignalNoiseRatio: 43}, status,
	)
	response = "[ 1687496917, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496919, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496921, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496922, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496923, 2609700, 177, 7064600, 849 ]\n" +
		"00:00:00:00:00:00  -52 dBm / -95 dBm (SNR 43)  0 ms ago\n" +
		"\tRX: 619.4 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(t, stationStatus{}, status)
}
