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

// Radio holds the current state of the access point's configuration and any robot radios connected to it.
type Radio struct {
	Channel                     int                       `json:"channel"`
	Status                      radioStatus               `json:"status"`
	StationStatuses             map[string]*StationStatus `json:"stationStatuses"`
	Type                        radioType                 `json:"-"`
	ConfigurationRequestChannel chan ConfigurationRequest `json:"-"`
	device                      string
	stationInterfaces           map[station]string
	uciTree                     uci.Tree
}

// radioType represents the hardware type of the radio.
//
//go:generate stringer -type=radioType
type radioType int

const (
	typeUnknown radioType = iota
	typeLinksys
	typeVividHosting
)

// radioStatus represents the configuration stage of the radio.
type radioStatus string

const (
	statusBooting     radioStatus = "BOOTING"
	statusConfiguring             = "CONFIGURING"
	statusActive                  = "ACTIVE"
	statusError                   = "ERROR"
)

var ssidRe = regexp.MustCompile("ESSID: \"([-\\w ]*)\"")

// NewRadio creates a new Radio instance and initializes its fields to default values.
func NewRadio() *Radio {
	radio := Radio{
		Status:                      statusBooting,
		ConfigurationRequestChannel: make(chan ConfigurationRequest, configurationRequestBufferSize),
		uciTree:                     uci.NewTree(uci.DefaultTreePath),
	}
	radio.determineAndSetType()
	if radio.Type == typeUnknown {
		log.Fatal("Unable to determine radio hardware type; exiting.")
	}
	log.Printf("Detected radio hardware type: %v", radio.Type)

	switch radio.Type {
	case typeLinksys:
		radio.device = "radio0"
		radio.stationInterfaces = map[station]string{
			red1:  "wlan0",
			red2:  "wlan0-1",
			red3:  "wlan0-2",
			blue1: "wlan0-3",
			blue2: "wlan0-4",
			blue3: "wlan0-5",
		}
	case typeVividHosting:
		radio.device = "wifi1"
		radio.stationInterfaces = map[station]string{
			red1:  "ath1",
			red2:  "ath11",
			red3:  "ath12",
			blue1: "ath13",
			blue2: "ath14",
			blue3: "ath15",
		}
	}

	radio.StationStatuses = make(map[string]*StationStatus)
	for i := 0; i < int(stationCount); i++ {
		radio.StationStatuses[station(i).String()] = nil
	}

	return &radio
}

// Run loops indefinitely, handling configuration requests and polling the Wi-Fi status.
func (radio *Radio) Run() {
	radio.waitForStartup()

	// Initialize the in-memory state to match the radio's current configuration.
	channel, _ := radio.uciTree.GetLast("wireless", radio.device, "channel")
	radio.Channel, _ = strconv.Atoi(channel)
	_ = radio.updateStationStatuses()
	radio.Status = statusActive

	for {
		// Check if there are any pending configuration requests; if not, periodically poll Wi-Fi status.
		select {
		case request := <-radio.ConfigurationRequestChannel:
			// If there are multiple requests queued up, only consider the latest one.
			numExtraRequests := len(radio.ConfigurationRequestChannel)
			for i := 0; i < numExtraRequests; i++ {
				request = <-radio.ConfigurationRequestChannel
			}
			radio.Status = statusConfiguring
			log.Printf("Processing configuration request: %+v", request)
			radio.configure(request)
			if len(radio.ConfigurationRequestChannel) == 0 {
				radio.Status = statusActive
			}
		case <-time.After(time.Second * statusPollIntervalSec):
			radio.updateStationMonitoring()
		}
	}
}

// determineAndSetType determines the model of the radio.
func (radio *Radio) determineAndSetType() {
	model, _ := radio.uciTree.GetLast("system", "@system[0]", "model")
	if strings.Contains(model, "VH") {
		radio.Type = typeVividHosting
	} else {
		radio.Type = typeLinksys
	}
}

// waitForStartup polls the Wi-Fi status and blocks until the radio has finished booting.
func (radio *Radio) waitForStartup() {
	for {
		if err := exec.Command("iwinfo", radio.stationInterfaces[red1], "info").Run(); err == nil {
			log.Println("Radio ready with baseline Wi-Fi configuration.")
			return
		}
		log.Println("Waiting for radio to finish starting up...")
		time.Sleep(bootPollIntervalSec * time.Second)
	}
}

// configure configures the radio with the given configuration.
func (radio *Radio) configure(request ConfigurationRequest) {
	if request.Channel > 0 {
		radio.uciTree.SetType("wireless", radio.device, "channel", uci.TypeOption, strconv.Itoa(request.Channel))
		radio.Channel = request.Channel
	}

	if radio.Type == typeLinksys {
		// Clear the state of the radio before loading teams; the Linksys AP is crash-prone otherwise.
		radio.configureStations(map[string]StationConfiguration{})
	}
	radio.configureStations(request.StationConfigurations)
}

// configureStations configures the access point with the given team station configurations.
func (radio *Radio) configureStations(stationConfigurations map[string]StationConfiguration) {
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
			radio.uciTree.SetType("wireless", wifiInterface, "ssid", uci.TypeOption, ssid)
			radio.uciTree.SetType("wireless", wifiInterface, "key", uci.TypeOption, wpaKey)
			if radio.Type == typeVividHosting {
				radio.uciTree.SetType("wireless", wifiInterface, "sae_password", uci.TypeOption, wpaKey)
			}

			if err := radio.uciTree.Commit(); err != nil {
				log.Printf("Failed to commit wireless configuration: %v", err)
			}
		}

		if err := exec.Command("wifi", "reload", radio.device).Run(); err != nil {
			log.Printf("Failed to reload configuration for device %s: %v", radio.device, err)
		}

		if radio.Type == typeLinksys {
			// The Linksys AP returns immediately after 'wifi reload' but may not have applied the configuration yet;
			// sleep for a bit to compensate. (The Vivid AP waits for the configuration to be applied before returning.)
			time.Sleep(time.Second * linksysWifiReloadBackoffSec)
		}
		err := radio.updateStationStatuses()
		if err != nil {
			log.Printf("Error updating station statuses: %v", err)
		} else if radio.stationSsidsAreCorrect(stationConfigurations) {
			log.Printf("Successfully configured Wi-Fi after %d attempts.", retryCount)
			break
		}

		log.Printf("Wi-Fi configuration still incorrect after %d attempts; trying again.", retryCount)
		retryCount++
	}
}

// updateStationStatuses fetches the current Wi-Fi status (SSID, WPA key, etc.) for each team station and updates the
// in-memory state.
func (radio *Radio) updateStationStatuses() error {
	for station, stationInterface := range radio.stationInterfaces {
		byteOutput, err := exec.Command("iwinfo", stationInterface, "info").Output()
		fmt.Printf("Output for iwinfo %s info: %s\n", stationInterface, string(byteOutput))
		if err != nil {
			return fmt.Errorf("error getting iwinfo for interface %s from AP: %v", stationInterface, err)
		} else {
			matches := ssidRe.FindStringSubmatch(string(byteOutput))
			if len(matches) > 0 {
				ssid := matches[1]
				if strings.HasPrefix(ssid, "no-team-") {
					radio.StationStatuses[station.String()] = nil
				} else {
					var status StationStatus
					status.Ssid = ssid
					status.HashedWpaKey, status.WpaKeySalt = radio.getHashedWpaKeyAndSalt(station)
					radio.StationStatuses[station.String()] = &status
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
func (radio *Radio) stationSsidsAreCorrect(stationConfigurations map[string]StationConfiguration) bool {
	for stationName, stationStatus := range radio.StationStatuses {
		if config, ok := stationConfigurations[stationName]; ok {
			if radio.StationStatuses[stationName].Ssid != config.Ssid {
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
func (radio *Radio) getHashedWpaKeyAndSalt(station station) (string, string) {
	wpaKey, ok := radio.uciTree.GetLast("wireless", fmt.Sprintf("@wifi-iface[%d]", station+1), "key")
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
func (radio *Radio) updateStationMonitoring() {
	for station, stationInterface := range radio.stationInterfaces {
		stationStatus := radio.StationStatuses[station.String()]
		if stationStatus == nil {
			// Skip stations that don't have a team assigned.
			continue
		}

		outputBytes, err := exec.Command("luci-bwc", "-i", stationInterface).Output()
		if err != nil {
			log.Printf("Error running 'luci-bwc -i %s': %v", stationInterface, err)
			stationStatus.BandwidthUsedMbps = monitoringErrorCode
		} else {
			stationStatus.parseBandwidthUsed(string(outputBytes))
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
