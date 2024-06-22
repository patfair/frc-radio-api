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

// Radio holds the current state of the access point's configuration and any robot radios connected to it.
type Radio struct {
	// 5GHz or 6GHz channel number the radio is broadcasting on.
	Channel int `json:"channel"`

	// Channel bandwidth mode for the radio to use. Valid values are "20MHz" and "40MHz".
	ChannelBandwidth string `json:"channelBandwidth"`

	// VLANs to use for the teams of the red alliance. Valid values are "10_20_30", "40_50_60", and "70_80_90".
	RedVlans AllianceVlans `json:"redVlans"`

	// VLANs to use for the teams of the blue alliance. Valid values are "10_20_30", "40_50_60", and "70_80_90".
	BlueVlans AllianceVlans `json:"blueVlans"`

	// Enum representing the current configuration stage of the radio.
	Status radioStatus `json:"status"`

	// Map of team station names to their current status.
	StationStatuses map[string]*NetworkStatus `json:"stationStatuses"`

	// Version of the radio software.
	Version string `json:"version"`

	// Queue for receiving and buffering configuration requests.
	ConfigurationRequestChannel chan ConfigurationRequest `json:"-"`

	// Hardware type of the radio.
	Type RadioType `json:"-"`

	// Name of the radio's Wi-Fi device, dependent on the hardware type.
	device string

	// Extra set of VLANs that are not used for team networks. Valid values are "10_20_30", "40_50_60", and "70_80_90".
	spareVlans AllianceVlans

	// List of Wi-Fi interface names in order of their corresponding VLAN, dependent on the hardware type.
	vlanInterfaces []string
}

// AllianceVlans represents which three VLANs are used for the teams of an alliance.
type AllianceVlans string

const (
	Vlans102030 AllianceVlans = "10_20_30"
	Vlans405060 AllianceVlans = "40_50_60"
	Vlans708090 AllianceVlans = "70_80_90"
)

// NewRadio creates a new Radio instance and initializes its fields to default values.
func NewRadio() *Radio {
	radio := Radio{
		RedVlans:                    Vlans102030,
		BlueVlans:                   Vlans405060,
		spareVlans:                  Vlans708090,
		Status:                      statusBooting,
		ConfigurationRequestChannel: make(chan ConfigurationRequest, configurationRequestBufferSize),
	}
	radio.determineAndSetType()
	if radio.Type == TypeUnknown {
		log.Fatal("Unable to determine radio hardware type; exiting.")
	}
	log.Printf("Detected radio hardware type: %v", radio.Type)
	radio.determineAndSetVersion()

	// Initialize the device and station interface names that are dependent on the hardware type.
	switch radio.Type {
	case TypeLinksys:
		radio.device = "radio0"
		radio.vlanInterfaces = []string{
			"wlan0",
			"wlan0-1",
			"wlan0-2",
			"wlan0-3",
			"wlan0-4",
			"wlan0-5",
			"wlan0-6",
			"wlan0-7",
			"wlan0-8",
		}
	case TypeVividHosting:
		radio.device = "wifi1"
		radio.vlanInterfaces = []string{
			"ath1",
			"ath11",
			"ath12",
			"ath13",
			"ath14",
			"ath15",
			"ath16",
			"ath17",
			"ath18",
		}
	}

	radio.StationStatuses = make(map[string]*NetworkStatus)
	for station := red1; station <= blue3; station++ {
		radio.StationStatuses[station.String()] = nil
	}

	return &radio
}

// getStationInterfaceIndex returns the Wi-Fi interface index for the given team station.
func (radio *Radio) getStationInterfaceIndex(station station) int {
	var vlans AllianceVlans
	var offset int
	if station == red1 || station == red2 || station == red3 {
		vlans = radio.RedVlans
		offset = int(station) - int(red1)
	} else if station == blue1 || station == blue2 || station == blue3 {
		vlans = radio.BlueVlans
		offset = int(station) - int(blue1)
	} else if station == spare1 || station == spare2 || station == spare3 {
		vlans = radio.spareVlans
		offset = int(station) - int(spare1)
	}

	switch vlans {
	case Vlans102030:
		return offset
	case Vlans405060:
		return 3 + offset
	case Vlans708090:
		return 6 + offset
	default:
		// Invalid station.
		return -1
	}
}

// getStationInterfaceName returns the Wi-Fi interface name for the given team station.
func (radio *Radio) getStationInterfaceName(station station) string {
	index := radio.getStationInterfaceIndex(station)
	if index == -1 {
		// Invalid station.
		return ""
	}
	return radio.vlanInterfaces[index]
}

// determineAndSetType determines the model of the radio.
func (radio *Radio) determineAndSetType() {
	model, _ := uciTree.GetLast("system", "@system[0]", "model")
	if strings.Contains(model, "VH") {
		radio.Type = TypeVividHosting
	} else {
		radio.Type = TypeLinksys
	}
}

// isStarted returns true if the Wi-Fi interface is up and running.
func (radio *Radio) isStarted() bool {
	_, err := shell.runCommand("iwinfo", radio.getStationInterfaceName(blue3), "info")
	return err == nil
}

// setInitialState initializes the in-memory state to match the radio's current configuration.
func (radio *Radio) setInitialState() {
	channel, _ := uciTree.GetLast("wireless", radio.device, "channel")
	radio.Channel, _ = strconv.Atoi(channel)
	htmode, _ := uciTree.GetLast("wireless", radio.device, "htmode")
	switch htmode {
	case "HT20":
		radio.ChannelBandwidth = "20MHz"
	case "HT40":
		radio.ChannelBandwidth = "40MHz"
	default:
		radio.ChannelBandwidth = "INVALID"
	}
	_ = radio.updateStationStatuses()
}

// configure configures the radio with the given configuration.
func (radio *Radio) configure(request ConfigurationRequest) error {
	if request.Channel > 0 {
		uciTree.SetType("wireless", radio.device, "channel", uci.TypeOption, strconv.Itoa(request.Channel))
		radio.Channel = request.Channel
	}
	if request.ChannelBandwidth != "" {
		var htmode string
		switch request.ChannelBandwidth {
		case "20MHz":
			htmode = "HT20"
		case "40MHz":
			htmode = "HT40"
		default:
			return fmt.Errorf("invalid channel bandwidth: %s", request.ChannelBandwidth)
		}
		uciTree.SetType("wireless", radio.device, "htmode", uci.TypeOption, htmode)
		radio.ChannelBandwidth = request.ChannelBandwidth
	}
	if request.RedVlans != "" && request.BlueVlans != "" {
		radio.RedVlans = request.RedVlans
		radio.BlueVlans = request.BlueVlans
		if radio.RedVlans != Vlans708090 && radio.BlueVlans != Vlans708090 {
			radio.spareVlans = Vlans708090
		} else if radio.RedVlans != Vlans405060 && radio.BlueVlans != Vlans405060 {
			radio.spareVlans = Vlans405060
		} else {
			radio.spareVlans = Vlans102030
		}
	}

	if radio.Type == TypeLinksys {
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
		for station := red1; station <= spare3; station++ {
			position := radio.getStationInterfaceIndex(station) + 1
			var ssid, wpaKey string
			if config, ok := stationConfigurations[station.String()]; ok {
				ssid = config.Ssid
				wpaKey = config.WpaKey
			} else {
				ssid = fmt.Sprintf("no-team-%d", position)
				wpaKey = ssid
			}

			wifiInterface := fmt.Sprintf("@wifi-iface[%d]", position)
			uciTree.SetType("wireless", wifiInterface, "ssid", uci.TypeOption, ssid)
			uciTree.SetType("wireless", wifiInterface, "key", uci.TypeOption, wpaKey)
			if radio.Type == TypeVividHosting {
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
	for station := red1; station <= blue3; station++ {
		ssid, err := getSsid(radio.getStationInterfaceName(station))
		if err != nil {
			return err
		}
		if strings.HasPrefix(ssid, "no-team-") {
			radio.StationStatuses[station.String()] = nil
		} else {
			var status NetworkStatus
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
	for station := red1; station <= blue3; station++ {
		stationStatus := radio.StationStatuses[station.String()]
		if stationStatus == nil {
			// Skip stations that don't have a team assigned.
			continue
		}

		stationStatus.updateMonitoring(radio.getStationInterfaceName(station))
	}
}
