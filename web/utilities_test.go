package web

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetVlan100IpAddress(t *testing.T) {
	ipAddress, err := getVlan100IpAddress()

	// Branch the test verification logic since it may or may not be Run on a system with a 10.0.100.x interface and
	// mocking the system calls to be deterministic is onerous.
	if err == nil {
		assert.Regexp(t, "^10\\.0\\.100\\.\\d+$", ipAddress)
	} else {
		assert.Contains(t, err.Error(), "no IP address found on VLAN 100")
	}
}
