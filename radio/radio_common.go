package radio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/digineo/go-uci"
	"log"
	"math/rand"
	"regexp"
	"time"
)

const (
	// How frequently to poll the radio while waiting for it to finish starting up.
	bootPollIntervalSec = 3

	// How frequently to poll the radio for its current status between configurations.
	monitoringPollIntervalSec = 5

	// How many configuration requests to buffer in memory.
	configurationRequestBufferSize = 10

	// How long to wait between retries when configuring the radio.
	retryBackoffSec = 3

	// Valid characters in the randomly generated salt used to obscure the WPA key.
	saltCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// Length of the randomly generated salt used to obscure the WPA key.
	saltLength = 16
)

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

var uciTree = uci.NewTree(uci.DefaultTreePath)
var shell shellWrapper = execShell{}
var ssidRe = regexp.MustCompile("ESSID: \"([-\\w ]*)\"")
var retryBackoffDuration = retryBackoffSec * time.Second

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