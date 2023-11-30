// This file is specific to the robot radio version of the API.
//go:build robot

package radio

import (
	"fmt"
	"github.com/digineo/go-uci"
	"log"
	"strconv"
	"time"
)

const (
	// Name of the radio's Wi-Fi device.
	radioDevice = "wifi1"

	// Name of the radio's Wi-Fi interface.
	radioInterface = "ath1"

	// Index of the radio's Wi-Fi interface section in the UCI configuration.
	radioInterfaceIndex = 0
)

// Radio holds the current state of the access point's configuration and any robot radios connected to it.
type Radio struct {
	// Team number that the radio is currently configured for.
	TeamNumber int `json:"teamNumber"`

	// Team-specific SSID.
	Ssid string `json:"ssid"`

	// SHA-256 hash of the WPA key and salt for the team, encoded as a hexadecimal string. The WPA key is not exposed
	// directly to prevent unauthorized users from learning its value. However, a user who already knows the WPA key can
	// verify that it is correct by concatenating it with the WpaKeySalt and hashing the result using SHA-256; the
	// result should match the HashedWpaKey.
	HashedWpaKey string `json:"hashedWpaKey"`

	// Randomly generated salt used to hash the WPA key.
	WpaKeySalt string `json:"wpaKeySalt"`

	// Enum representing the current configuration stage of the radio.
	Status radioStatus `json:"status"`

	// Queue for receiving and buffering configuration requests.
	ConfigurationRequestChannel chan ConfigurationRequest `json:"-"`
}

// NewRadio creates a new Radio instance and initializes its fields to default values.
func NewRadio() *Radio {
	return &Radio{
		Status:                      statusBooting,
		ConfigurationRequestChannel: make(chan ConfigurationRequest, configurationRequestBufferSize),
	}
}

// isStarted returns true if the Wi-Fi interface is up and running.
func (radio *Radio) isStarted() bool {
	_, err := shell.runCommand("iwinfo", radioInterface, "info")
	return err == nil
}

// setInitialState initializes the in-memory state to match the radio's current configuration.
func (radio *Radio) setInitialState() {
	radio.Ssid, _ = uciTree.GetLast("wireless", fmt.Sprintf("@wifi-iface[%d]", radioInterfaceIndex), "ssid")
	radio.TeamNumber, _ = strconv.Atoi(radio.Ssid)
	radio.HashedWpaKey, radio.WpaKeySalt = radio.getHashedWpaKeyAndSalt(radioInterfaceIndex)
}

// configure configures the radio with the given configuration.
func (radio *Radio) configure(request ConfigurationRequest) error {
	retryCount := 1

	for {
		// Handle Wi-Fi.
		ssid := strconv.Itoa(request.TeamNumber)
		wifiInterface := fmt.Sprintf("@wifi-iface[%d]", radioInterfaceIndex)
		uciTree.SetType("wireless", wifiInterface, "ssid", uci.TypeOption, ssid)
		uciTree.SetType("wireless", wifiInterface, "key", uci.TypeOption, request.WpaKey)

		// Handle DHCP.
		teamPartialIp := fmt.Sprintf("%d.%d", request.TeamNumber/100, request.TeamNumber%100)
		uciTree.SetType("dhcp", "lan", "dhcp_option", uci.TypeList, fmt.Sprintf("3,10.%s.4", teamPartialIp))
		uciTree.SetType("dhcp", "@host[0]", "name", uci.TypeOption, fmt.Sprintf("roboRIO-%d-FRC", request.TeamNumber))
		uciTree.SetType("dhcp", "@host[0]", "ip", uci.TypeOption, fmt.Sprintf("10.%s.2", teamPartialIp))

		// Handle IP address.
		uciTree.SetType("network", "lan", "ipaddr", uci.TypeOption, fmt.Sprintf("10.%s.1", teamPartialIp))
		uciTree.SetType("network", "lan", "gateway", uci.TypeOption, fmt.Sprintf("10.%s.4", teamPartialIp))

		if err := uciTree.Commit(); err != nil {
			return fmt.Errorf("failed to commit configuration: %v", err)
		}
		if _, err := shell.runCommand("wifi", "reload", radioDevice); err != nil {
			return fmt.Errorf("failed to reload Wi-Fi configuration for device %s: %v", radioDevice, err)
		}

		var err error
		radio.Ssid, err = getSsid(radioInterface)
		if err != nil {
			return err
		}
		radio.TeamNumber, _ = strconv.Atoi(radio.Ssid)
		radio.HashedWpaKey, radio.WpaKeySalt = radio.getHashedWpaKeyAndSalt(radioInterfaceIndex)
		if radio.TeamNumber == request.TeamNumber {
			log.Printf("Successfully configured robot radio after %d attempts.", retryCount)
			break
		}

		log.Printf("Wi-Fi configuration still incorrect after %d attempts; trying again.", retryCount)
		time.Sleep(retryBackoffDuration)
		retryCount++
	}

	return nil
}

// updateMonitoring is a no-op for the robot radio, for the time being, since the API is only used for
// one-time-per-event configuration.
func (radio *Radio) updateMonitoring() {
}
