package radio

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStationStatus_ParseBandwithUsed(t *testing.T) {
	var status StationStatus

	// Response is too short.
	status.parseBandwidthUsed("")
	assert.Equal(t, 0.0, status.BandwidthUsedMbps)
	response := "[ 1687496957, 26097, 177, 71670, 865 ],\n" +
		"[ 1687496958, 26097, 177, 71734, 866 ],\n" +
		"[ 1687496959, 26097, 177, 71734, 866 ],\n" +
		"[ 1687496960, 26097, 177, 71798, 867 ],\n" +
		"[ 1687496960, 26097, 177, 71798, 867 ],\n" +
		"[ 1687496961, 26097, 177, 71798, 867 ]"
	status.parseBandwidthUsed(response)
	assert.Equal(t, 0.0, status.BandwidthUsedMbps)

	// Response is normal.
	response = "[ 1687496917, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496919, 26097, 177, 70454, 846 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496920, 26097, 177, 70518, 847 ],\n" +
		"[ 1687496921, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496922, 26097, 177, 70582, 848 ],\n" +
		"[ 1687496923, 2609700, 177, 7064600, 849 ]"
	status.parseBandwidthUsed(response)
	assert.Equal(t, 15.324, status.BandwidthUsedMbps)
}

func TestStationStatus_ParseAssocList(t *testing.T) {
	var status StationStatus

	status.parseAssocList("")
	assert.Equal(t, StationStatus{}, status)

	// MAC address is invalid.
	response := "00:00:00:00:00:00  -53 dBm / -95 dBm (SNR 42)  0 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(t, StationStatus{}, status)

	// Link is valid.
	response = "48:DA:35:B0:00:CF  -53 dBm / -95 dBm (SNR 42)  0 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 254.0 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(
		t, StationStatus{IsRobotRadioLinked: true, RxRateMbps: 550.6, TxRateMbps: 254.0, SignalNoiseRatio: 42}, status,
	)
	response = "48:DA:35:B0:00:CF  -53 dBm / -95 dBm (SNR 7)  4000 ms ago\n" +
		"\tRX: 123.4 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(
		t, StationStatus{IsRobotRadioLinked: true, RxRateMbps: 123.4, TxRateMbps: 550.6, SignalNoiseRatio: 7}, status,
	)

	// Link is stale.
	response = "48:DA:35:B0:00:CF  -53 dBm / -95 dBm (SNR 42)  4001 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                   0 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(t, StationStatus{}, status)
}
