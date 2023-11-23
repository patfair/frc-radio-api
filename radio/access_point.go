package radio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/digineo/go-uci"
	"log"
	"math/rand"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	bootPollIntervalSec            = 3
	statusPollIntervalSec          = 5
	configurationRequestBufferSize = 10
	linksysWifiReloadBackoffSec    = 5
	saltCharacters                 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	saltLength                     = 16
	monitoringErrorCode            = -999
)

// AccessPoint holds the current state of the access point's configuration and any robot radios connected to it.
type AccessPoint struct {
	Channel                     int                       `json:"channel"`
	Status                      accessPointStatus         `json:"status"`
	StationStatuses             map[string]*StationStatus `json:"stationStatuses"`
	Type                        accessPointType           `json:"-"`
	ConfigurationRequestChannel chan ConfigurationRequest `json:"-"`
	device                      string
	stationInterfaces           map[station]string
}

// accessPointType represents the hardware type of the access point.
//
//go:generate stringer -type=accessPointType
type accessPointType int

const (
	typeUnknown accessPointType = iota
	typeLinksys
	typeVividHosting
)

// accessPointStatus represents the configuration stage of the access point.
type accessPointStatus string

const (
	statusBooting     accessPointStatus = "BOOTING"
	statusConfiguring                   = "CONFIGURING"
	statusActive                        = "ACTIVE"
	statusError                         = "ERROR"
)

// StationStatus encapsulates the status of a single team station on the access point.
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

// station represents an alliance and position to which a team is assigned.
//
//go:generate stringer -type=station
type station int

const (
	red1 station = iota
	red2
	red3
	blue1
	blue2
	blue3
	stationCount
)

var ssidRe = regexp.MustCompile("ESSID: \"([-\\w ]*)\"")

// newAccessPoint creates a new access point instance and initializes its fields to default values.
func NewAccessPoint() *AccessPoint {
	ap := AccessPoint{
		Status:                      statusBooting,
		Type:                        determineHardwareType(),
		ConfigurationRequestChannel: make(chan ConfigurationRequest, configurationRequestBufferSize),
	}
	if ap.Type == typeUnknown {
		log.Fatal("Unable to determine access point hardware type; exiting.")
	}
	log.Printf("Detected access point hardware type: %v", ap.Type)

	switch ap.Type {
	case typeLinksys:
		ap.device = "radio0"
		ap.stationInterfaces = map[station]string{
			red1:  "wlan0",
			red2:  "wlan0-1",
			red3:  "wlan0-2",
			blue1: "wlan0-3",
			blue2: "wlan0-4",
			blue3: "wlan0-5",
		}
	case typeVividHosting:
		ap.device = "wifi1"
		ap.stationInterfaces = map[station]string{
			red1:  "ath1",
			red2:  "ath11",
			red3:  "ath12",
			blue1: "ath13",
			blue2: "ath14",
			blue3: "ath15",
		}
	}

	ap.StationStatuses = make(map[string]*StationStatus)
	for i := 0; i < int(stationCount); i++ {
		ap.StationStatuses[station(i).String()] = nil
	}

	return &ap
}

// run loops indefinitely, handling configuration requests and polling the Wi-Fi status.
func (ap *AccessPoint) Run() {
	ap.waitForStartup()

	// Initialize the in-memory state to match the access point's current configuration.
	channel, _ := uci.GetLast("wireless", ap.device, "channel")
	ap.Channel, _ = strconv.Atoi(channel)
	_ = ap.updateStationStatuses()
	ap.Status = statusActive

	for {
		// Check if there are any pending configuration requests; if not, periodically poll Wi-Fi status.
		select {
		case request := <-ap.ConfigurationRequestChannel:
			// If there are multiple requests queued up, only consider the latest one.
			numExtraRequests := len(ap.ConfigurationRequestChannel)
			for i := 0; i < numExtraRequests; i++ {
				request = <-ap.ConfigurationRequestChannel
			}
			ap.Status = statusConfiguring
			log.Printf("Processing configuration request: %+v", request)
			ap.configure(request)
			if len(ap.ConfigurationRequestChannel) == 0 {
				ap.Status = statusActive
			}
		case <-time.After(time.Second * statusPollIntervalSec):
			ap.updateStationMonitoring()
		}
	}
}

// determineHardwareType determines the model of the access point.
func determineHardwareType() accessPointType {
	model, _ := uci.GetLast("system", "@system[0]", "model")
	if strings.Contains(model, "VH") {
		return typeVividHosting
	}
	return typeLinksys
}

// waitForStartup polls the Wi-Fi status and blocks until the access point has finished booting.
func (ap *AccessPoint) waitForStartup() {
	for {
		if err := exec.Command("iwinfo", ap.stationInterfaces[red1], "info").Run(); err == nil {
			log.Println("Access point ready with baseline Wi-Fi configuration.")
			return
		}
		log.Println("Waiting for access point to finish starting up...")
		time.Sleep(bootPollIntervalSec * time.Second)
	}
}

// configure configures the access point with the given configuration.
func (ap *AccessPoint) configure(request ConfigurationRequest) {
	if request.Channel > 0 {
		uci.Set("wireless", ap.device, "channel", strconv.Itoa(request.Channel))
		ap.Channel = request.Channel
	}

	if ap.Type == typeLinksys {
		// Clear the state of the radio before loading teams; the Linksys AP is crash-prone otherwise.
		ap.configureStations(map[string]StationConfiguration{})
	}
	ap.configureStations(request.StationConfigurations)
}

// configureStations configures the access point with the given team station configurations.
func (ap *AccessPoint) configureStations(stationConfigurations map[string]StationConfiguration) {
	retryCount := 1

	for {
		for stationIndex := 0; stationIndex < 6; stationIndex++ {
			position := stationIndex + 1
			var ssid, wpaKey string
			if config, ok := stationConfigurations[station(stationIndex).String()]; ok {
				ssid = config.Ssid
				wpaKey = config.WpaKey
			} else {
				ssid = fmt.Sprintf("no-team-%d", position)
				wpaKey = ssid
			}

			wifiInterface := fmt.Sprintf("@wifi-iface[%d]", position)
			uci.Set("wireless", wifiInterface, "ssid", ssid)
			uci.Set("wireless", wifiInterface, "key", wpaKey)
			if ap.Type == typeVividHosting {
				uci.Set("wireless", wifiInterface, "sae_password", wpaKey)
			}

			if err := uci.Commit(); err != nil {
				log.Printf("Failed to commit wireless configuration: %v", err)
			}
		}

		if err := exec.Command("wifi", "reload", ap.device).Run(); err != nil {
			log.Printf("Failed to reload configuration for device %s: %v", ap.device, err)
		}

		if ap.Type == typeLinksys {
			// The Linksys AP returns immediately after 'wifi reload' but may not have applied the configuration yet;
			// sleep for a bit to compensate. (The Vivid AP waits for the configuration to be applied before returning.)
			time.Sleep(time.Second * linksysWifiReloadBackoffSec)
		}
		err := ap.updateStationStatuses()
		if err != nil {
			log.Printf("Error updating station statuses: %v", err)
		} else if ap.stationSsidsAreCorrect(stationConfigurations) {
			log.Printf("Successfully configured Wi-Fi after %d attempts.", retryCount)
			break
		}

		log.Printf("Wi-Fi configuration still incorrect after %d attempts; trying again.", retryCount)
		retryCount++
	}
}

// updateStationStatuses fetches the current Wi-Fi status (SSID, WPA key, etc.) for each team station and updates the
// in-memory state.
func (ap *AccessPoint) updateStationStatuses() error {
	for station, stationInterface := range ap.stationInterfaces {
		byteOutput, err := exec.Command("iwinfo", stationInterface, "info").Output()
		fmt.Printf("Output for iwinfo %s info: %s\n", stationInterface, string(byteOutput))
		if err != nil {
			return fmt.Errorf("error getting iwinfo for interface %s from AP: %v", stationInterface, err)
		} else {
			matches := ssidRe.FindStringSubmatch(string(byteOutput))
			if len(matches) > 0 {
				ssid := matches[1]
				if strings.HasPrefix(ssid, "no-team-") {
					ap.StationStatuses[station.String()] = nil
				} else {
					var status StationStatus
					status.Ssid = ssid
					status.HashedWpaKey, status.WpaKeySalt = getHashedWpaKeyAndSalt(station)
					ap.StationStatuses[station.String()] = &status
				}
			} else {
				return fmt.Errorf(
					"error parsing iwinfo output for interface %s from AP: \n%s", stationInterface, string(byteOutput),
				)
			}
		}
	}

	return nil
}

// stationSsidsAreCorrect returns true if the configured networks as read from the access point match the requested
// configuration.
func (ap *AccessPoint) stationSsidsAreCorrect(stationConfigurations map[string]StationConfiguration) bool {
	for stationName, stationStatus := range ap.StationStatuses {
		if config, ok := stationConfigurations[stationName]; ok {
			if ap.StationStatuses[stationName].Ssid != config.Ssid {
				return false
			}
		} else {
			if stationStatus != nil {
				// This is an error case; we expect the station status to be nil if the station is not configured.
				return false
			}
		}
	}

	return true
}

// getHashedWpaKeyAndSalt fetches the WPA key for the given station and returns its hashed value and the salt used for
// hashing.
func getHashedWpaKeyAndSalt(station station) (string, string) {
	wpaKey, ok := uci.GetLast("wireless", fmt.Sprintf("@wifi-iface[%d]", station+1), "key")
	if !ok {
		return "", ""
	}
	// Generate a random string of 16 characters to use as the salt.
	saltBytes := make([]byte, saltLength)
	for i := 0; i < saltLength; i++ {
		saltBytes[i] = saltCharacters[rand.Intn(len(saltCharacters))]
	}
	salt := string(saltBytes)
	hash := sha256.New()
	hash.Write([]byte(wpaKey + salt))
	hashedWpaKey := hex.EncodeToString(hash.Sum(nil))

	return hashedWpaKey, salt
}

// updateStationMonitoring polls the access point for the current bandwidth usage and link state of each team station
// and updates the in-memory state.
func (ap *AccessPoint) updateStationMonitoring() {
	for station, stationInterface := range ap.stationInterfaces {
		stationStatus := ap.StationStatuses[station.String()]
		if stationStatus == nil {
			// Skip stations that don't have a team assigned.
			continue
		}

		outputBytes, err := exec.Command("luci-bwc", "-i", stationInterface).Output()
		if err != nil {
			log.Printf("Error running 'luci-bwc -i %s': %v", stationInterface, err)
			stationStatus.BandwidthUsedMbps = monitoringErrorCode
		} else {
			stationStatus.BandwidthUsedMbps = parseBandwidthUsed(string(outputBytes))
		}
		outputBytes, err = exec.Command("iwinfo", stationInterface, "assoclist").Output()
		if err != nil {
			log.Printf("Error running 'iwinfo %s assoclist': %v", stationInterface, err)
			stationStatus.RxRateMbps = monitoringErrorCode
			stationStatus.TxRateMbps = monitoringErrorCode
			stationStatus.SignalNoiseRatio = monitoringErrorCode
		} else {
			stationStatus.parseAssocList(string(outputBytes))
		}
	}
}

// parseBandwidthUsed parses the given data from the access point's onboard bandwidth monitor and returns five-second
// average bandwidth in megabits per second.
func parseBandwidthUsed(response string) float64 {
	mBits := 0.0
	btuRe := regexp.MustCompile("\\[ (\\d+), (\\d+), (\\d+), (\\d+), (\\d+) ]")
	btuMatches := btuRe.FindAllStringSubmatch(response, -1)
	if len(btuMatches) >= 7 {
		firstMatch := btuMatches[len(btuMatches)-6]
		lastMatch := btuMatches[len(btuMatches)-1]
		rXBytes, _ := strconv.Atoi(lastMatch[2])
		tXBytes, _ := strconv.Atoi(lastMatch[4])
		rXBytesOld, _ := strconv.Atoi(firstMatch[2])
		tXBytesOld, _ := strconv.Atoi(firstMatch[4])
		mBits = float64(rXBytes-rXBytesOld+tXBytes-tXBytesOld) * 0.000008 / 5.0
	}
	return mBits
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
