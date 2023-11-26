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

	// Upper-bound link receive rate (from the robot radio to the access point) in megabits per second.
	RxRateMbps float64 `json:"rxRateMbps"`

	// Upper-bound link transmit rate (from the access point to the robot radio) in megabits per second.
	TxRateMbps float64 `json:"txRateMbps"`

	// Current signal-to-noise ratio (SNR) in decibels.
	SignalNoiseRatio int `json:"signalNoiseRatio"`

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

// Parses the given data from the access point's association list and updates the status structure with the result.
func (status *StationStatus) parseAssocList(response string) {
	radioLinkRe := regexp.MustCompile("((?:[0-9A-F]{2}:){5}(?:[0-9A-F]{2})).*\\(SNR (\\d+)\\)\\s+(\\d+) ms ago")
	rxRateRe := regexp.MustCompile("RX:\\s+(\\d+\\.\\d+)\\s+MBit/s")
	txRateRe := regexp.MustCompile("TX:\\s+(\\d+\\.\\d+)\\s+MBit/s")

	status.IsRobotRadioLinked = false
	status.RxRateMbps = 0
	status.TxRateMbps = 0
	status.SignalNoiseRatio = 0
	for _, radioLinkMatch := range radioLinkRe.FindAllStringSubmatch(response, -1) {
		macAddress := radioLinkMatch[1]
		dataAgeMs, _ := strconv.Atoi(radioLinkMatch[3])
		if macAddress != "00:00:00:00:00:00" && dataAgeMs <= 4000 {
			status.IsRobotRadioLinked = true
			status.SignalNoiseRatio, _ = strconv.Atoi(radioLinkMatch[2])
			rxRateMatch := rxRateRe.FindStringSubmatch(response)
			if len(rxRateMatch) > 0 {
				status.RxRateMbps, _ = strconv.ParseFloat(rxRateMatch[1], 64)
			}
			txRateMatch := txRateRe.FindStringSubmatch(response)
			if len(txRateMatch) > 0 {
				status.TxRateMbps, _ = strconv.ParseFloat(txRateMatch[1], 64)
			}
			break
		}
	}
}
