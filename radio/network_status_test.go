package radio

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNetworkStatus_ParseBandwidthUsed(t *testing.T) {
	var status NetworkStatus

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

func TestNetworkStatus_ParseAssocList(t *testing.T) {
	var status NetworkStatus

	status.parseAssocList("")
	assert.Equal(t, NetworkStatus{}, status)

	// MAC address is invalid.
	response := "00:00:00:00:00:00  -53 dBm / -95 dBm (SNR 42)  0 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                 123 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(t, NetworkStatus{}, status)

	// Link is valid.
	response = "48:DA:35:B0:00:CF  -53 dBm / -95 dBm (SNR 42)  0 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 254.0 MBit/s                                 123 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(
		t,
		NetworkStatus{
			IsLinked:          true,
			MacAddress:        "48:DA:35:B0:00:CF",
			SignalDbm:         -53,
			NoiseDbm:          -95,
			SignalNoiseRatio:  42,
			RxRateMbps:        550.6,
			RxPackets:         4095,
			TxRateMbps:        254.0,
			TxPackets:         123,
			ConnectionQuality: "excellent",
		},
		status,
	)
	response = "37:DA:35:B0:00:BE  -64 dBm / -84 dBm (SNR 7)  4000 ms ago\n" +
		"\tRX: 123.4 MBit/s                                5091 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                 789 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(
		t,
		NetworkStatus{
			IsLinked:          true,
			MacAddress:        "37:DA:35:B0:00:BE",
			SignalDbm:         -64,
			NoiseDbm:          -84,
			SignalNoiseRatio:  7,
			RxRateMbps:        123.4,
			RxPackets:         5091,
			TxRateMbps:        550.6,
			TxPackets:         789,
			ConnectionQuality: "warning",
		},
		status,
	)

	// Link is stale.
	response = "48:DA:35:B0:00:CF  -53 dBm / -95 dBm (SNR 42)  4001 ms ago\n" +
		"\tRX: 550.6 MBit/s                                4095 Pkts.\n" +
		"\tTX: 550.6 MBit/s                                 123 Pkts.\n" +
		"\texpected throughput: unknown"
	status.parseAssocList(response)
	assert.Equal(t, NetworkStatus{}, status)
}

func TestNetworkStatus_ParseIfconfig(t *testing.T) {
	var status NetworkStatus

	status.parseIfconfig("")
	assert.Equal(t, NetworkStatus{}, status)

	response := "ath15\tLink encap:Ethernet  HWaddr 4A:DA:35:B0:00:2C\n" +
		"\tinet6 addr: fe80::48da:35ff:feb0:2c/64 Scope:Link\n" +
		"\tUP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1\n" +
		"\tRX packets:690 errors:0 dropped:0 overruns:0 frame:0\n" +
		"\tTX packets:727 errors:0 dropped:0 overruns:0 carrier:0\n " +
		"\tcollisions:0 txqueuelen:0\n" +
		"\tRX bytes:45311 (44.2 KiB)  TX bytes:48699 (47.5 KiB)\n"
	status.parseIfconfig(response)
	assert.Equal(t, NetworkStatus{RxBytes: 45311, TxBytes: 48699}, status)
}

func TestNetworkStatus_DetermineConnectionQuality(t *testing.T) {
	var status NetworkStatus

	assert.Equal(t, NetworkStatus{}, status)

	status.RxRateMbps = connectionQualityExcellentMinimum
	status.determineConnectionQuality()
	assert.Equal(t, "excellent", status.ConnectionQuality)

	status.RxRateMbps = connectionQualityGoodMinimum
	status.determineConnectionQuality()
	assert.Equal(t, "good", status.ConnectionQuality)

	status.RxRateMbps = connectionQualityCautionMinimum
	status.determineConnectionQuality()
	assert.Equal(t, "caution", status.ConnectionQuality)

	status.RxRateMbps = 0.1
	status.determineConnectionQuality()
	assert.Equal(t, "warning", status.ConnectionQuality)

	// Ensure ConnectionQuality resets to blank.
	status.parseAssocList("")
	assert.Equal(t, NetworkStatus{}, status)
}
