package radio

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTriggerFirmwareUpdate(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeTree.valuesForGet["system.@system[0].model"] = ""
	fakeShell := newFakeShell(t)
	shell = fakeShell

	// Success case.
	fakeShell.commandOutput["sysupgrade -n some-file"] = "some output"
	TriggerFirmwareUpdate("some-file")
	assert.Equal(t, 1, len(fakeShell.commandsRun))
	assert.Contains(t, fakeShell.commandsRun, "sysupgrade -n some-file")

	// Error case.
	fakeShell.reset()
	fakeShell.commandErrors["sysupgrade -n some-file"] = errors.New("oops")
	TriggerFirmwareUpdate("some-file")
	assert.Equal(t, 1, len(fakeShell.commandsRun))
	assert.Contains(t, fakeShell.commandsRun, "sysupgrade -n some-file")

	// Check that LED is blinked for the Vivid-Hosting radio.
	fakeTree.valuesForGet["system.@system[0].model"] = "VH-109(AP)"
	fakeShell.reset()
	fakeShell.commandOutput["sysupgrade -n some-file"] = "some output"
	fakeShell.commandOutput["sh -c kill $(ps | grep fms_check.sh | grep -v grep | awk '{print $1}')"] = ""
	fakeShell.commandOutput["sh -c echo timer > /sys/class/leds/sys/trigger"] = ""
	fakeShell.commandOutput["sh -c echo 50 > /sys/class/leds/sys/delay_on && echo 50 > /sys/class/leds/sys/delay_off"] =
		""
	TriggerFirmwareUpdate("some-file")
	assert.Equal(t, 4, len(fakeShell.commandsRun))
	assert.Contains(t, fakeShell.commandsRun, "sh -c kill $(ps | grep fms_check.sh | grep -v grep | awk '{print $1}')")
	assert.Contains(t, fakeShell.commandsRun, "sh -c echo timer > /sys/class/leds/sys/trigger")
	assert.Contains(
		t,
		fakeShell.commandsRun,
		"sh -c echo 50 > /sys/class/leds/sys/delay_on && echo 50 > /sys/class/leds/sys/delay_off",
	)
}

func TestRadio_determineAndSetVersion(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeShell := newFakeShell(t)
	shell = fakeShell

	// Vivid-Hosting success case.
	fakeTree.valuesForGet["system.@system[0].model"] = "VH-109(AP)"
	fakeShell.commandOutput["cat /etc/vh_firmware"] = "\tVH version 1.2.3 \n"
	radio := Radio{}
	radio.determineAndSetVersion()
	assert.Equal(t, "VH version 1.2.3", radio.Version)

	// Vivid-Hosting error case.
	fakeTree.reset()
	fakeTree.valuesForGet["system.@system[0].model"] = "VH-109(AP)"
	fakeShell.reset()
	fakeShell.commandErrors["cat /etc/vh_firmware"] = errors.New("oops")
	radio = Radio{}
	radio.determineAndSetVersion()
	assert.Equal(t, "unknown", radio.Version)

	// Linksys success case.
	fakeTree.reset()
	fakeTree.valuesForGet["system.@system[0].model"] = ""
	fakeShell.commandOutput["sh -c source /etc/openwrt_release && echo $DISTRIB_DESCRIPTION"] = "\tLinksys v2.3.4 \n"
	radio = Radio{}
	radio.determineAndSetVersion()
	assert.Equal(t, "Linksys v2.3.4", radio.Version)

	// Linksys error case.
	fakeTree.reset()
	fakeTree.valuesForGet["system.@system[0].model"] = ""
	fakeShell.reset()
	fakeShell.commandErrors["sh -c source /etc/openwrt_release && echo $DISTRIB_DESCRIPTION"] = errors.New("oops")
	radio = Radio{}
	radio.determineAndSetVersion()
	assert.Equal(t, "unknown", radio.Version)
}
