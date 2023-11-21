package main

// accessPoint holds the current state of the access point's configuration and any robot radios connected to it.
type accessPoint struct {
	Status          accessPointStatus         `json:"status"`
	StationStatuses map[string]*stationStatus `json:"stationStatuses"`
}

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
	stationStatuses := make(map[string]*stationStatus)
	for i := 0; i < int(stationCount); i++ {
		stationStatuses[station(i).String()] = nil
	}

	return &accessPoint{
		Status:          statusBooting,
		StationStatuses: stationStatuses,
	}
}
