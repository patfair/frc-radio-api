// This file is specific to the access point version of the API.
//go:build !robot

package radio

import (
	"fmt"
	"github.com/digineo/go-uci"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	// Sentinel value used to populate status fields when a monitoring command failed.
	monitoringErrorCode = -999
)

// Radio holds the current state of the access point's configuration and any robot radios connected to it.
type Radio struct {
	// 5GHz or 6GHz channel number the radio is broadcasting on.
	Channel int `json:"channel"`

	// Enum representing the current configuration stage of the radio.
	Status radioStatus `json:"status"`

	// Map of team station names to their current status.
	StationStatuses map[string]*StationStatus `json:"stationStatuses"`

	// Queue for receiving and buffering configuration requests.
	ConfigurationRequestChannel chan ConfigurationRequest `json:"-"`

	// Hardware type of the radio.
	Type radioType `json:"-"`

	// Name of the radio's Wi-Fi device, dependent on the hardware type.
	device string

	// Map of team station names to their Wi-Fi interface names, dependent on the hardware type.
	stationInterfaces map[station]string
}

// NewRadio creates a new Radio instance and initializes its fields to default values.
func NewRadio() *Radio {
	radio := Radio{
		Status:                      statusBooting,
		ConfigurationRequestChannel: make(chan ConfigurationRequest, configurationRequestBufferSize),
	}
	radio.determineAndSetType()
	if radio.Type == typeUnknown {
		log.Fatal("Unable to determine radio hardware type; exiting.")
	}
	log.Printf("Detected radio hardware type: %v", radio.Type)

	// Initialize the device and station interface names that are dependent on the hardware type.
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

// determineAndSetType determines the model of the radio.
func (radio *Radio) determineAndSetType() {
	model, _ := uciTree.GetLast("system", "@system[0]", "model")
	if strings.Contains(model, "VH") {
		radio.Type = typeVividHosting
	} else {
		radio.Type = typeLinksys
	}
}

// isStarted returns true if the Wi-Fi interface is up and running.
func (radio *Radio) isStarted() bool {
	_, err := shell.runCommand("iwinfo", radio.stationInterfaces[blue3], "info")
	return err == nil
}

// setInitialState initializes the in-memory state to match the radio's current configuration.
func (radio *Radio) setInitialState() {
	channel, _ := uciTree.GetLast("wireless", radio.device, "channel")
	radio.Channel, _ = strconv.Atoi(channel)
	_ = radio.updateStationStatuses()
}

// configure configures the radio with the given configuration.
func (radio *Radio) configure(request ConfigurationRequest) error {
	if request.Channel > 0 {
		uciTree.SetType("wireless", radio.device, "channel", uci.TypeOption, strconv.Itoa(request.Channel))
		radio.Channel = request.Channel
	}

	if radio.Type == typeLinksys {
		// Clear the state of the radio before loading teams; the Linksys AP is crash-prone otherwise.
		if err := radio.configureStations(map[string]StationConfiguration{}); err != nil {
			return err
		}
		time.Sleep(wifiReloadBackoffDuration)
	}
	return radio.configureStations(request.StationConfigurations)
}

// configureStations configures the access point with the given team station configurations.
func (radio *Radio) configureStations(stationConfigurations map[string]StationConfiguration) error {
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
			uciTree.SetType("wireless", wifiInterface, "ssid", uci.TypeOption, ssid)
			uciTree.SetType("wireless", wifiInterface, "key", uci.TypeOption, wpaKey)
			if radio.Type == typeVividHosting {
				uciTree.SetType("wireless", wifiInterface, "sae_password", uci.TypeOption, wpaKey)
			}

			if err := uciTree.Commit(); err != nil {
				return fmt.Errorf("failed to commit wireless configuration: %v", err)
			}
		}

		if _, err := shell.runCommand("wifi", "reload", radio.device); err != nil {
			return fmt.Errorf("failed to reload configuration for device %s: %v", radio.device, err)
		}
		time.Sleep(wifiReloadBackoffDuration)

		err := radio.updateStationStatuses()
		if err != nil {
			return fmt.Errorf("error updating station statuses: %v", err)
		} else if radio.stationSsidsAreCorrect(stationConfigurations) {
			log.Printf("Successfully configured Wi-Fi after %d attempts.", retryCount)
			break
		}

		log.Printf("Wi-Fi configuration still incorrect after %d attempts; trying again.", retryCount)
		time.Sleep(retryBackoffDuration)
		retryCount++
	}

	return nil
}

// updateStationStatuses fetches the current Wi-Fi status (SSID, WPA key, etc.) for each team station and updates the
// in-memory state.
func (radio *Radio) updateStationStatuses() error {
	for station, stationInterface := range radio.stationInterfaces {
		ssid, err := getSsid(stationInterface)
		if err != nil {
			return err
		}
		if strings.HasPrefix(ssid, "no-team-") {
			radio.StationStatuses[station.String()] = nil
		} else {
			var status StationStatus
			status.Ssid = ssid
			status.HashedWpaKey, status.WpaKeySalt = radio.getHashedWpaKeyAndSalt(int(station) + 1)
			radio.StationStatuses[station.String()] = &status
		}
	}

	return nil
}

// stationSsidsAreCorrect returns true if the configured networks as read from the access point match the requested
// configuration.
func (radio *Radio) stationSsidsAreCorrect(stationConfigurations map[string]StationConfiguration) bool {
	for stationName, stationStatus := range radio.StationStatuses {
		if config, ok := stationConfigurations[stationName]; ok {
			if radio.StationStatuses[stationName] == nil || radio.StationStatuses[stationName].Ssid != config.Ssid {
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

// updateMonitoring polls the access point for the current bandwidth usage and link state of each team station and
// updates the in-memory state.
func (radio *Radio) updateMonitoring() {
	for station, stationInterface := range radio.stationInterfaces {
		stationStatus := radio.StationStatuses[station.String()]
		if stationStatus == nil {
			// Skip stations that don't have a team assigned.
			continue
		}

		// Update the bandwidth usage.
		output, err := shell.runCommand("luci-bwc", "-i", stationInterface)
		if err != nil {
			log.Printf("Error running 'luci-bwc -i %s': %v", stationInterface, err)
			stationStatus.BandwidthUsedMbps = monitoringErrorCode
		} else {
			stationStatus.parseBandwidthUsed(output)
		}

		// Update the link state of any associated robot radios.
		output, err = shell.runCommand("iwinfo", stationInterface, "assoclist")
		if err != nil {
			log.Printf("Error running 'iwinfo %s assoclist': %v", stationInterface, err)
			stationStatus.RxRateMbps = monitoringErrorCode
			stationStatus.TxRateMbps = monitoringErrorCode
			stationStatus.SignalNoiseRatio = monitoringErrorCode
		} else {
			stationStatus.parseAssocList(output)
		}
	}
}
