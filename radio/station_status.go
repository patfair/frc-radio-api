// This file is specific to the access point version of the API.
//go:build !robot

package radio

import (
	"math"
	"regexp"
	"strconv"
)

// StationStatus encapsulates the status of a single team station on the q point.
type StationStatus struct {
	// Team-specific SSID for the station, usually equal to the team number as a string.
	Ssid string `json:"ssid"`

	// SHA-256 hash of the WPA key and salt for the station, encoded as a hexadecimal string. The WPA key is not exposed
	// directly to prevent unauthorized users from learning its value. However, a user who already knows the WPA key can
	// verify that it is correct by concatenating it with the WpaKeySalt and hashing the result using SHA-256; the
	// result should match the HashedWpaKey.
	HashedWpaKey string `json:"hashedWpaKey"`

	// Randomly generated salt used to hash the WPA key.
	WpaKeySalt string `json:"wpaKeySalt"`

	// Whether a robot radio is currently associated to the access point on this station's SSID.
	IsRobotRadioLinked bool `json:"isRobotRadioLinked"`

	// MAC address of the robot radio currently associated to the access point on this station's SSID.
	MacAddress string `json:"macAddress"`

	// Signal strength of the robot radio's link to the access point, in decibel-milliwatts.
	SignalDbm int `json:"signalDbm"`

	// Noise level of the robot radio's link to the access point, in decibel-milliwatts.
	NoiseDbm int `json:"noiseDbm"`

	// Current signal-to-noise ratio (SNR) in decibels.
	SignalNoiseRatio int `json:"signalNoiseRatio"`

	// Upper-bound link receive rate (from the robot radio to the access point) in megabits per second.
	RxRateMbps float64 `json:"rxRateMbps"`

	// Number of packets received from the robot radio.
	RxPackets int `json:"rxPackets"`

	// Number of bytes received from the robot radio.
	RxBytes int `json:"rxBytes"`

	// Upper-bound link transmit rate (from the access point to the robot radio) in megabits per second.
	TxRateMbps float64 `json:"txRateMbps"`

	// Number of packets transmitted to the robot radio.
	TxPackets int `json:"txPackets"`

	// Number of bytes transmitted to the robot radio.
	TxBytes int `json:"txBytes"`

	// Current five-second average total (rx + tx) bandwidth in megabits per second.
	BandwidthUsedMbps float64 `json:"bandwidthUsedMbps"`
}

// parseBandwidthUsed parses the given data from the access point's onboard bandwidth monitor and returns five-second
// average bandwidth in megabits per second.
func (status *StationStatus) parseBandwidthUsed(response string) {
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

// parseAssocList parses the given data from the access point's association list and updates the status structure with
// the result.
func (status *StationStatus) parseAssocList(response string) {
	line1Re := regexp.MustCompile(
		"((?:[0-9A-F]{2}:){5}(?:[0-9A-F]{2}))\\s+(-\\d+) dBm / (-\\d+) dBm \\(SNR (\\d+)\\)\\s+(\\d+) ms ago",
	)
	line2Re := regexp.MustCompile("RX:\\s+(\\d+\\.\\d+)\\s+MBit/s\\s+(\\d+) Pkts.")
	line3R3 := regexp.MustCompile("TX:\\s+(\\d+\\.\\d+)\\s+MBit/s\\s+(\\d+) Pkts.")

	status.IsRobotRadioLinked = false
	status.MacAddress = ""
	status.SignalDbm = 0
	status.NoiseDbm = 0
	status.SignalNoiseRatio = 0
	status.RxRateMbps = 0
	status.RxPackets = 0
	status.TxRateMbps = 0
	status.TxPackets = 0
	for _, line1Match := range line1Re.FindAllStringSubmatch(response, -1) {
		macAddress := line1Match[1]
		dataAgeMs, _ := strconv.Atoi(line1Match[5])
		if macAddress != "00:00:00:00:00:00" && dataAgeMs <= 4000 {
			status.IsRobotRadioLinked = true
			status.MacAddress = macAddress
			status.SignalDbm, _ = strconv.Atoi(line1Match[2])
			status.NoiseDbm, _ = strconv.Atoi(line1Match[3])
			status.SignalNoiseRatio, _ = strconv.Atoi(line1Match[4])
			line2Match := line2Re.FindStringSubmatch(response)
			if len(line2Match) > 0 {
				status.RxRateMbps, _ = strconv.ParseFloat(line2Match[1], 64)
				status.RxPackets, _ = strconv.Atoi(line2Match[2])
			}
			line3Match := line3R3.FindStringSubmatch(response)
			if len(line3Match) > 0 {
				status.TxRateMbps, _ = strconv.ParseFloat(line3Match[1], 64)
				status.TxPackets, _ = strconv.Atoi(line3Match[2])
			}
			break
		}
	}
}

// parseIfconfig parses the given output from the access point's ifconfig command and updates the status structure with
// the result.
func (status *StationStatus) parseIfconfig(response string) {
	bytesRe := regexp.MustCompile("RX bytes:(\\d+) .* TX bytes:(\\d+) ")

	status.RxBytes = 0
	status.TxBytes = 0
	bytesMatch := bytesRe.FindStringSubmatch(response)
	if len(bytesMatch) > 0 {
		status.RxBytes, _ = strconv.Atoi(bytesMatch[1])
		status.TxBytes, _ = strconv.Atoi(bytesMatch[2])
	}
}
