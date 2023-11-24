package radio

import (
	"regexp"
	"strconv"
)

// StationStatus encapsulates the status of a single team station on the q point.
type StationStatus struct {
	Ssid               string  `json:"ssid"`
	HashedWpaKey       string  `json:"hashedWpaKey"`
	WpaKeySalt         string  `json:"wpaKeySalt"`
	IsRobotRadioLinked bool    `json:"isRobotRadioLinked"`
	RxRateMbps         float64 `json:"rxRateMbps"`
	TxRateMbps         float64 `json:"txRateMbps"`
	SignalNoiseRatio   int     `json:"signalNoiseRatio"`
	BandwidthUsedMbps  float64 `json:"bandwidthUsedMbps"`
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
		status.BandwidthUsedMbps = float64(rXBytes-rXBytesOld+tXBytes-tXBytesOld) * 0.000008 / 5.0
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
