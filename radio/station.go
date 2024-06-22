package radio

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
	spare1
	spare2
	spare3
)
