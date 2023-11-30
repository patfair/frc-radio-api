// This file is specific to the robot radio version of the API.
//go:build robot

package web

import (
	"fmt"
)

// getListenAddress returns the address and port that the web server should listen on.
func getListenAddress() string {
	return fmt.Sprintf(":%d", port)
}
