package main

import (
	"github.com/digineo/go-uci"
	"log"
	"os/exec"
	"strings"
	"time"
)

const (
	bootPollIntervalSec = 3
)

// accessPoint holds the current state of the access point's configuration and any robot radios connected to it.
type accessPoint struct {
	Status            accessPointStatus         `json:"status"`
	StationStatuses   map[string]*stationStatus `json:"stationStatuses"`
	hardwareType      accessPointType
	stationInterfaces map[station]string
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

// stationStatus encapsulates the status of a single team station on the access point.
type stationStatus struct {
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

// newAccessPoint creates a new access point instance and initializes its fields to default values.
func newAccessPoint() *accessPoint {
	ap := accessPoint{
		Status:       statusBooting,
		hardwareType: determineHardwareType(),
	}
	if ap.hardwareType == typeUnknown {
		log.Fatal("Unable to determine access point hardware type; exiting.")
	}
	log.Printf("Detected access point hardware type: %v", ap.hardwareType)

	switch ap.hardwareType {
	case typeLinksys:
		ap.stationInterfaces = map[station]string{
			red1:  "wlan0",
			red2:  "wlan0-1",
			red3:  "wlan0-2",
			blue1: "wlan0-3",
			blue2: "wlan0-4",
			blue3: "wlan0-5",
		}
	case typeVividHosting:
		ap.stationInterfaces = map[station]string{
			red1:  "ath1",
			red2:  "ath11",
			red3:  "ath12",
			blue1: "ath13",
			blue2: "ath14",
			blue3: "ath15",
		}
	}

	ap.StationStatuses = make(map[string]*stationStatus)
	for i := 0; i < int(stationCount); i++ {
		ap.StationStatuses[station(i).String()] = nil
	}

	return &ap
}

// run loops indefinitely, handling configuration requests and polling the Wi-Fi status.
func (ap *accessPoint) run() {
	ap.waitForStartup()

	for {
		time.Sleep(time.Second)
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
func (ap *accessPoint) waitForStartup() {
	for {
		if err := exec.Command("iwinfo", ap.stationInterfaces[red1], "info").Run(); err == nil {
			log.Println("Access point ready with baseline Wi-Fi configuration.")
			ap.Status = statusActive
			return
		}
		log.Println("Waiting for access point to finish starting up...")
		time.Sleep(bootPollIntervalSec * time.Second)
	}
}
