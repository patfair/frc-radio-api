package radio

import (
	"log"
	"math"
	"regexp"
	"strconv"
)

const (
	// Sentinel value used to populate status fields when a monitoring command failed.
	monitoringErrorCode = -999

	// Cutoff values used to determine the connection quality of the interface based on RX rate.
	connectionQualityExcellentMinimum = 412.9
	connectionQualityGoodMinimum      = 309.7
	connectionQualityCautionMinimum   = 172.1
)

// NetworkStatus encapsulates the status of a single Wi-Fi interface on the device (i.e. a team SSID network on the
// access point or one of the two interfaces on the robot radio).
type NetworkStatus struct {
	// SSID for the network.
	Ssid string `json:"ssid"`

	// SHA-256 hash of the WPA key and salt for the network, encoded as a hexadecimal string. The WPA key is not exposed
	// directly to prevent unauthorized users from learning its value. However, a user who already knows the WPA key can
	// verify that it is correct by concatenating it with the WpaKeySalt and hashing the result using SHA-256; the
	// result should match the HashedWpaKey.
	HashedWpaKey string `json:"hashedWpaKey"`

	// Randomly generated salt used to hash the WPA key.
	WpaKeySalt string `json:"wpaKeySalt"`

	// Whether this network is currently associated with a remote device.
	IsLinked bool `json:"isLinked"`

	// MAC address of the remote device currently associated with this network. Blank if not associated.
	MacAddress string `json:"macAddress"`

	// Signal strength of the link to the remote device, in decibel-milliwatts. Zero if not associated.
	SignalDbm int `json:"signalDbm"`

	// Noise level of the link to the remote device, in decibel-milliwatts. Zero if not associated.
	NoiseDbm int `json:"noiseDbm"`

	// Current signal-to-noise ratio (SNR) in decibels. Zero if not associated.
	SignalNoiseRatio int `json:"signalNoiseRatio"`

	// Upper-bound link receive rate (from the remote device to this one) in megabits per second. Zero if not
	// associated.
	RxRateMbps float64 `json:"rxRateMbps"`

	// Number of packets received from the remote device. Zero if not associated.
	RxPackets int `json:"rxPackets"`

	// Number of bytes received from the remote device. Zero if not associated.
	RxBytes int `json:"rxBytes"`

	// Upper-bound link transmit rate (from this device to the remote one) in megabits per second. Zero if not
	// associated.
	TxRateMbps float64 `json:"txRateMbps"`

	// Number of packets transmitted to the remote device. Zero if not associated.
	TxPackets int `json:"txPackets"`

	// Number of bytes transmitted to the remote device. Zero if not associated.
	TxBytes int `json:"txBytes"`

	// Current five-second average total (rx + tx) bandwidth in megabits per second.
	BandwidthUsedMbps float64 `json:"bandwidthUsedMbps"`

	// Human-readable string describing connection quality to the remote device. Based on RX rate. Blank if not associated.
	ConnectionQuality string `json:"connectionQuality"`

	// Flag representing whether the interface is for a robot.
	IsRobot bool `json:"-"`
}

// updateMonitoring polls the access point for the current bandwidth usage and link state of the given network interface
// and updates the in-memory state.
func (status *NetworkStatus) updateMonitoring(networkInterface string) {
	// Update the bandwidth usage.
	output, err := shell.runCommand("luci-bwc", "-i", networkInterface)
	if err != nil {
		log.Printf("Error running 'luci-bwc -i %s': %v", networkInterface, err)
		status.BandwidthUsedMbps = monitoringErrorCode
	} else {
		status.parseBandwidthUsed(output)
	}

	// Update the link state of any associated robot radios.
	output, err = shell.runCommand("iwinfo", networkInterface, "assoclist")
	if err != nil {
		log.Printf("Error running 'iwinfo %s assoclist': %v", networkInterface, err)
		status.RxRateMbps = monitoringErrorCode
		status.TxRateMbps = monitoringErrorCode
		status.SignalNoiseRatio = monitoringErrorCode
	} else {
		status.parseAssocList(output)
	}

	// Update the number of bytes received and transmitted.
	output, err = shell.runCommand("ifconfig", networkInterface)
	if err != nil {
		log.Printf("Error running 'ifconfig %s': %v", networkInterface, err)
		status.RxBytes = monitoringErrorCode
		status.TxBytes = monitoringErrorCode
	} else {
		status.parseIfconfig(output)
	}
}

// parseBandwidthUsed parses the given data from the radio's onboard bandwidth monitor and returns five-second average
// bandwidth in megabits per second.
func (status *NetworkStatus) parseBandwidthUsed(response string) {
	status.BandwidthUsedMbps = 0.0
	btuRe := regexp.MustCompile("\\[ (\\d+), (\\d+), (\\d+), (\\d+), (\\d+) ]")
	btuMatches := btuRe.FindAllStringSubmatch(response, -1)
	if len(btuMatches) >= 7 {
		firstMatch := btuMatches[len(btuMatches)-6]
		lastMatch := btuMatches[len(btuMatches)-1]
		rXBytes, _ := strconv.Atoi(lastMatch[2])
		tXBytes, _ := strconv.Atoi(lastMatch[4])
		rXBytesOld, _ := strconv.Atoi(firstMatch[2])
		tXBytesOld, _ := strconv.Atoi(firstMatch[4])
		status.BandwidthUsedMbps = math.Round(1000*float64(rXBytes-rXBytesOld+tXBytes-tXBytesOld)*0.000008/5.0) / 1000
	}
}

// parseAssocList parses the given data from the radio's association list and updates the status structure with the
// result.
func (status *NetworkStatus) parseAssocList(response string) {
	line1Re := regexp.MustCompile(
		"((?:[0-9A-F]{2}:){5}(?:[0-9A-F]{2}))\\s+(-\\d+) dBm / (-\\d+) dBm \\(SNR (\\d+)\\)\\s+(\\d+) ms ago",
	)
	line2Re := regexp.MustCompile("RX:\\s+(\\d+\\.\\d+)\\s+MBit/s\\s+(\\d+) Pkts.")
	line3R3 := regexp.MustCompile("TX:\\s+(\\d+\\.\\d+)\\s+MBit/s\\s+(\\d+) Pkts.")

	status.IsLinked = false
	status.MacAddress = ""
	status.SignalDbm = 0
	status.NoiseDbm = 0
	status.SignalNoiseRatio = 0
	status.RxRateMbps = 0
	status.RxPackets = 0
	status.TxRateMbps = 0
	status.TxPackets = 0
	status.ConnectionQuality = ""
	for _, line1Match := range line1Re.FindAllStringSubmatch(response, -1) {
		macAddress := line1Match[1]
		dataAgeMs, _ := strconv.Atoi(line1Match[5])
		if macAddress != "00:00:00:00:00:00" && dataAgeMs <= 4000 {
			status.IsLinked = true
			status.MacAddress = macAddress
			status.SignalDbm, _ = strconv.Atoi(line1Match[2])
			status.NoiseDbm, _ = strconv.Atoi(line1Match[3])
			status.SignalNoiseRatio, _ = strconv.Atoi(line1Match[4])
			line2Match := line2Re.FindStringSubmatch(response)
			if len(line2Match) > 0 {
				status.RxRateMbps, _ = strconv.ParseFloat(line2Match[1], 64)
				status.RxPackets, _ = strconv.Atoi(line2Match[2])
				if !status.IsRobot {
					status.determineConnectionQuality(status.RxRateMbps)
				}
			}
			line3Match := line3R3.FindStringSubmatch(response)
			if len(line3Match) > 0 {
				status.TxRateMbps, _ = strconv.ParseFloat(line3Match[1], 64)
				status.TxPackets, _ = strconv.Atoi(line3Match[2])
				if status.IsRobot {
					status.determineConnectionQuality(status.TxRateMbps)
				}
			}
			break
		}
	}
}

// parseIfconfig parses the given output from the radio's ifconfig command and updates the status structure with the
// result.
func (status *NetworkStatus) parseIfconfig(response string) {
	bytesRe := regexp.MustCompile("RX bytes:(\\d+) .* TX bytes:(\\d+) ")

	status.RxBytes = 0
	status.TxBytes = 0
	bytesMatch := bytesRe.FindStringSubmatch(response)
	if len(bytesMatch) > 0 {
		status.RxBytes, _ = strconv.Atoi(bytesMatch[1])
		status.TxBytes, _ = strconv.Atoi(bytesMatch[2])
	}
}

// determineConnectionQuality uses the stored RxRateMbps value to determine a connection quality string and updates the
// status structure with the result.
func (status *NetworkStatus) determineConnectionQuality(rate float64) {
	if rate >= connectionQualityExcellentMinimum {
		status.ConnectionQuality = "excellent"
	} else if rate >= connectionQualityGoodMinimum {
		status.ConnectionQuality = "good"
	} else if rate >= connectionQualityCautionMinimum {
		status.ConnectionQuality = "caution"
	} else {
		status.ConnectionQuality = "warning"
	}
}
