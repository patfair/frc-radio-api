// This file is specific to the robot radio version of the API.
//go:build robot

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
	// Name of the radio's 2.4GHz Wi-Fi device.
	radioDevice24 = "wifi0"

	// Name of the radio's 2.4GHz Wi-Fi interface.
	radioInterface24 = "ath0"

	// Index of the radio's 2.4GHz Wi-Fi interface section in the UCI configuration.
	radioInterfaceIndex24 = 0

	// Name of the radio's 6GHz Wi-Fi device.
	radioDevice6 = "wifi1"

	// Name of the radio's 6GHz Wi-Fi interface.
	radioInterface6 = "ath1"

	// Index of the radio's 6GHz Wi-Fi interface section in the UCI configuration.
	radioInterfaceIndex6 = 1

	// Seperator between the team number and SSID suffix in the Wi-Fi SSIDs.
	ssidSuffixSeperator = "-"
)

// Radio holds the current state of the access point's configuration and any robot radios connected to it.
type Radio struct {
	// Operation mode that the radio is currently configured for.
	Mode radioMode `json:"mode"`

	// 6GHz channel number the radio is broadcasting on, if configured to TEAM_ACCESS_POINT mode.
	Channel string `json:"channel"`

	// Team number that the radio is currently configured for.
	TeamNumber int `json:"teamNumber"`

	// Suffix currently appended to the 6GHz network SSID.
	SsidSuffix string `json:"ssidSuffix"`

	// Status of the radio's 2.4GHz network.
	NetworkStatus24 NetworkStatus `json:"networkStatus24"`

	// Status of the radio's 6GHz network.
	NetworkStatus6 NetworkStatus `json:"networkStatus6"`

	// Enum representing the current configuration stage of the radio.
	Status radioStatus `json:"status"`

	// Version of the radio software.
	Version string `json:"version"`

	// Queue for receiving and buffering configuration requests.
	ConfigurationRequestChannel chan ConfigurationRequest `json:"-"`
}

// radioMode represents the configuration mode of the radio.
type radioMode string

const (
	// The radio is configured as a Wi-Fi client and connects to an access point.
	modeTeamRobotRadio radioMode = "TEAM_ROBOT_RADIO"

	// The radio is configured as an access point and provides Wi-Fi to robot radios and other devices such as computers
	// used in programming robots.
	modeTeamAccessPoint radioMode = "TEAM_ACCESS_POINT"
)

// NewRadio creates a new Radio instance and initializes its fields to default values.
func NewRadio() *Radio {
	radio := Radio{
		Status:                      statusBooting,
		ConfigurationRequestChannel: make(chan ConfigurationRequest, configurationRequestBufferSize),
	}
	radio.determineAndSetVersion()

	return &radio
}

// isStarted returns true if the Wi-Fi interface is up and running.
func (radio *Radio) isStarted() bool {
	_, err := shell.runCommand("iwinfo", radioInterface6, "info")
	return err == nil
}

// setInitialState initializes the in-memory state to match the radio's current configuration.
func (radio *Radio) setInitialState() {
	wifiInterface24 := fmt.Sprintf("@wifi-iface[%d]", radioInterfaceIndex24)
	wifiInterface6 := fmt.Sprintf("@wifi-iface[%d]", radioInterfaceIndex6)
	mode, _ := uciTree.GetLast("wireless", wifiInterface6, "mode")
	if mode == "sta" {
		radio.Mode = modeTeamRobotRadio
		radio.Channel = ""
		radio.NetworkStatus24.IsRobot = true
		radio.NetworkStatus6.IsRobot = true
	} else {
		radio.Mode = modeTeamAccessPoint
		radio.Channel, _ = uciTree.GetLast("wireless", radioDevice6, "channel")
	}

	radio.NetworkStatus24.Ssid, _ = uciTree.GetLast("wireless", wifiInterface24, "ssid")
	radio.NetworkStatus24.HashedWpaKey, radio.NetworkStatus24.WpaKeySalt =
		radio.getHashedWpaKeyAndSalt(radioInterfaceIndex24)
	radio.NetworkStatus6.Ssid, _ = uciTree.GetLast("wireless", wifiInterface6, "ssid")
	radio.NetworkStatus6.HashedWpaKey, radio.NetworkStatus6.WpaKeySalt =
		radio.getHashedWpaKeyAndSalt(radioInterfaceIndex6)
	teamNumber, suffix, _ := strings.Cut(radio.NetworkStatus6.Ssid, ssidSuffixSeperator)
	radio.TeamNumber, _ = strconv.Atoi(teamNumber)
	radio.SsidSuffix = suffix
}

// configure configures the radio with the given configuration.
func (radio *Radio) configure(request ConfigurationRequest) error {
	retryCount := 1

	for {
		radio.Mode = request.Mode

		// Handle Wi-Fi.
		var ssid string
		if len(request.SsidSuffix) > 0 {
			ssid = fmt.Sprintf("%d%s%s", request.TeamNumber, ssidSuffixSeperator, request.SsidSuffix)
		} else {
			ssid = strconv.Itoa(request.TeamNumber)
		}
		wifiInterface6 := fmt.Sprintf("@wifi-iface[%d]", radioInterfaceIndex6)
		wifiInterface24 := fmt.Sprintf("@wifi-iface[%d]", radioInterfaceIndex24)
		uciTree.SetType("wireless", wifiInterface6, "ssid", uci.TypeOption, ssid)
		uciTree.SetType("wireless", wifiInterface6, "key", uci.TypeOption, request.WpaKey6)

		teamPartialIp := fmt.Sprintf("%d.%d", request.TeamNumber/100, request.TeamNumber%100)
		if request.Mode == modeTeamRobotRadio {
			uciTree.SetType("wireless", wifiInterface6, "mode", uci.TypeOption, "sta")
			uciTree.SetType(
				"wireless", wifiInterface24, "ssid", uci.TypeOption, fmt.Sprintf("FRC-%s", ssid),
			)
			uciTree.SetType("wireless", wifiInterface24, "key", uci.TypeOption, request.WpaKey24)
			uciTree.SetType("wireless", wifiInterface24, "mode", uci.TypeOption, "ap")

			radio.Channel = ""
			uciTree.Del("wireless", radioDevice6, "channel")
			uciTree.SetType("wireless", radioDevice24, "channel", uci.TypeOption, "auto")
			uciTree.SetType("wireless", radioDevice24, "disabled", uci.TypeOption, "0")

			// Handle IP address when in STA mode.
			uciTree.SetType("network", "lan", "ipaddr", uci.TypeOption, fmt.Sprintf("10.%s.1", teamPartialIp))
			uciTree.SetType("network", "lan", "gateway", uci.TypeOption, fmt.Sprintf("10.%s.4", teamPartialIp))
			uciTree.SetType("dhcp", "lan", "start", uci.TypeOption, "200")
			uciTree.SetType("dhcp", "lan", "limit", uci.TypeOption, "20")

			// Handle NetworkStatus as robot.
			radio.NetworkStatus24.IsRobot = true
			radio.NetworkStatus6.IsRobot = true
		} else {
			uciTree.SetType("wireless", wifiInterface6, "mode", uci.TypeOption, "ap")

			uciTree.SetType("wireless", radioDevice24, "disabled", uci.TypeOption, "1")
			if request.Channel == 0 {
				radio.Channel = "auto"
				uciTree.SetType("wireless", radioDevice6, "channel", uci.TypeOption, "auto")
			} else {
				radio.Channel = strconv.Itoa(request.Channel)
				uciTree.SetType("wireless", radioDevice6, "channel", uci.TypeOption, strconv.Itoa(request.Channel))
			}

			// Handle IP address when in AP mode.
			uciTree.SetType("network", "lan", "ipaddr", uci.TypeOption, fmt.Sprintf("10.%s.4", teamPartialIp))
			uciTree.SetType("network", "lan", "gateway", uci.TypeOption, fmt.Sprintf("10.%s.4", teamPartialIp))
			uciTree.SetType("dhcp", "lan", "start", uci.TypeOption, "20")
			uciTree.SetType("dhcp", "lan", "limit", uci.TypeOption, "180")

			// Handle NetworkStatus as AP
			radio.NetworkStatus24.IsRobot = false
			radio.NetworkStatus24.IsRobot = false
		}

		// Handle DHCP.
		uciTree.DelSection("dhcp", "@host[-1]")
		uciTree.AddSection("dhcp", "@host[0]", "")
		uciTree.SetType("dhcp", "lan", "dhcp_option", uci.TypeList, fmt.Sprintf("3,10.%s.4", teamPartialIp))
		uciTree.SetType("dhcp", "@host[0]", "name", uci.TypeOption, fmt.Sprintf("roboRIO-%d-FRC", request.TeamNumber))
		uciTree.SetType("dhcp", "@host[0]", "ip", uci.TypeOption, fmt.Sprintf("10.%s.2", teamPartialIp))

		if err := uciTree.Commit(); err != nil {
			return fmt.Errorf("failed to commit configuration: %v", err)
		}
		if _, err := shell.runCommand("wifi", "reload"); err != nil {
			return fmt.Errorf("failed to reload Wi-Fi configuration: %v", err)
		}
		time.Sleep(wifiReloadBackoffDuration)

		var err error
		radio.NetworkStatus6.Ssid, err = getSsid(radioInterface6)
		if err != nil {
			return err
		}
		teamNumber, suffix, _ := strings.Cut(radio.NetworkStatus6.Ssid, ssidSuffixSeperator)
		radio.TeamNumber, _ = strconv.Atoi(teamNumber)
		radio.SsidSuffix = suffix
		radio.NetworkStatus6.HashedWpaKey, radio.NetworkStatus6.WpaKeySalt =
			radio.getHashedWpaKeyAndSalt(radioInterfaceIndex6)
		if radio.TeamNumber == request.TeamNumber && radio.SsidSuffix == request.SsidSuffix {
			log.Printf("Successfully configured robot radio after %d attempts.", retryCount)
			break
		}

		log.Printf("Wi-Fi configuration still incorrect after %d attempts; trying again.", retryCount)
		time.Sleep(retryBackoffDuration)
		retryCount++
	}

	return nil
}

// updateMonitoring polls the access point for the current bandwidth usage and link state of each network and updates
// the in-memory state.
func (radio *Radio) updateMonitoring() {
	radio.NetworkStatus6.updateMonitoring(radioInterface6)
	radio.NetworkStatus24.updateMonitoring(radioInterface24)
}
