package radio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/digineo/go-uci"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

const (
	// How frequently to poll the radio while waiting for it to finish starting up.
	bootPollIntervalSec = 3

	// How frequently to poll the radio for its current status between configurations.
	monitoringPollIntervalSec = 5

	// How long to wait after reloading the Wi-Fi configuration before polling the status.
	wifiReloadBackoffSec = 5

	// How many configuration requests to buffer in memory.
	configurationRequestBufferSize = 10

	// How long to wait between retries when configuring the radio.
	retryBackoffSec = 3

	// Minimum length for WPA keys.
	minWpaKeyLength = 8

	// Maximum length for WPA keys.
	maxWpaKeyLength = 16

	// Valid characters in the randomly generated salt used to obscure the WPA key.
	saltCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Length of the randomly generated salt used to obscure the WPA key.
	saltLength = 16

	// Regex to validate the a string as alphanumeric.
	alphanumericRegex = "^[a-zA-Z0-9]*$"
)

// RadioType represents the hardware type of the radio.
//
//go:generate stringer -type=RadioType
type RadioType int

const (
	TypeUnknown RadioType = iota
	TypeLinksys
	TypeVividHosting
)

// radioStatus represents the configuration stage of the radio.
type radioStatus string

const (
	statusBooting     radioStatus = "BOOTING"
	statusConfiguring radioStatus = "CONFIGURING"
	statusActive      radioStatus = "ACTIVE"
	statusError       radioStatus = "ERROR"
)

var uciTree = uci.NewTree(uci.DefaultTreePath)
var shell shellWrapper = execShell{}
var ssidRe = regexp.MustCompile("ESSID: \"([-\\w ]*)\"")
var retryBackoffDuration = retryBackoffSec * time.Second
var wifiReloadBackoffDuration = wifiReloadBackoffSec * time.Second

// Run loops indefinitely, handling configuration requests and polling the Wi-Fi status.
func (radio *Radio) Run() {
	for !radio.isStarted() {
		log.Println("Waiting for radio to finish starting up...")
		time.Sleep(bootPollIntervalSec * time.Second)
	}
	log.Println("Radio ready.")

	radio.setInitialState()
	radio.Status = statusActive

	for {
		// Check if there are any pending configuration requests; if not, periodically poll Wi-Fi status.
		select {
		case request := <-radio.ConfigurationRequestChannel:
			_ = radio.handleConfigurationRequest(request)
		case <-time.After(monitoringPollIntervalSec * time.Second):
			radio.updateMonitoring()
		}
	}
}

// TriggerFirmwareUpdate initiates the firmware update process using the given firmware file. This method may not return
// cleanly even if successful since the update utility will terminate this process.
func TriggerFirmwareUpdate(firmwarePath string) {
	log.Printf("Attempting to trigger firmware update using %s", firmwarePath)

	// Blink the SYS LED to indicate that we're loading firmware.
	model, _ := uciTree.GetLast("system", "@system[0]", "model")
	if strings.Contains(model, "VH") {
		_, _ = shell.runCommand("sh", "-c", "kill $(ps | grep fms_check.sh | grep -v grep | awk '{print $1}')")
		_, _ = shell.runCommand("sh", "-c", "echo timer > /sys/class/leds/sys/trigger")
		_, _ = shell.runCommand(
			"sh", "-c", "echo 50 > /sys/class/leds/sys/delay_on && echo 50 > /sys/class/leds/sys/delay_off",
		)
	}

	if err := shell.startCommand("sysupgrade", "-n", firmwarePath); err != nil {
		log.Printf("Error running sysupgrade: %v", err)
	}
	log.Println("Started sysupgrade successfully.")
}

// determineAndSetVersion determines the firmware version of the radio.
func (radio *Radio) determineAndSetVersion() {
	model, _ := uciTree.GetLast("system", "@system[0]", "model")
	var version string
	var err error
	if strings.Contains(model, "VH") {
		version, err = shell.runCommand("cat", "/etc/vh_firmware")
	} else {
		version, err = shell.runCommand("sh", "-c", "source /etc/openwrt_release && echo $DISTRIB_DESCRIPTION")
	}
	if err != nil {
		log.Printf("Error determining firmware version: %v", err)
		radio.Version = "unknown"
	} else {
		radio.Version = strings.TrimSpace(version)
	}
}

func (radio *Radio) handleConfigurationRequest(request ConfigurationRequest) error {
	// If there are multiple requests queued up, only consider the latest one.
	numExtraRequests := len(radio.ConfigurationRequestChannel)
	for i := 0; i < numExtraRequests; i++ {
		request = <-radio.ConfigurationRequestChannel
	}

	radio.Status = statusConfiguring
	log.Printf("Processing configuration request: %+v", request)
	if err := radio.configure(request); err != nil {
		log.Printf("Error configuring radio: %v", err)
		radio.Status = statusError
		return err
	} else if len(radio.ConfigurationRequestChannel) == 0 {
		radio.Status = statusActive
	}
	return nil
}

// getHashedWpaKeyAndSalt fetches the WPA key for the given station and returns its hashed value and the salt used for
// hashing.
func (radio *Radio) getHashedWpaKeyAndSalt(position int) (string, string) {
	wpaKey, ok := uciTree.GetLast("wireless", fmt.Sprintf("@wifi-iface[%d]", position), "key")
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

// getSsid fetches the post-configuration SSID of the given Wi-Fi interface using 'iwinfo info'.
func getSsid(wifiInterface string) (string, error) {
	output, err := shell.runCommand("iwinfo", wifiInterface, "info")
	if err != nil {
		return "", fmt.Errorf("error getting iwinfo for interface %s: %v", wifiInterface, err)
	} else {
		matches := ssidRe.FindStringSubmatch(output)
		if len(matches) > 0 {
			return matches[1], nil
		} else {
			return "", fmt.Errorf("error parsing iwinfo output for interface %s: %s", wifiInterface, output)
		}
	}
}

// isValid6GhzChannel returns true if the given channel is a valid 6GHz channel.
func isValid6GhzChannel(channel int) bool {
	x := (channel - 5) / 8
	y := (channel - 5) % 8
	return y == 0 && x >= 0 && x <= 28
}
