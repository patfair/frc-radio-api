package main

type accessPoint struct {
	Status          accessPointStatus
	StationStatuses map[string]*stationStatus
}

type accessPointStatus string

const (
	statusBooting     accessPointStatus = "BOOTING"
	statusConfiguring                   = "CONFIGURING"
	statusActive                        = "ACTIVE"
	statusError                         = "ERROR"
)

type stationStatus struct {
	TeamId             int
	HashedWpaKey       string
	WpaKeySalt         string
	IsRobotRadioLinked bool
	RxRateMbps         float64
	TxRateMbps         float64
	SignalNoiseRatio   int
	BandwidthUsedMbps  float64
}

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
