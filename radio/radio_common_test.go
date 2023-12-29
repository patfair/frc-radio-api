package radio

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTriggerFirmwareUpdate(t *testing.T) {
	fakeShell := newFakeShell(t)
	shell = fakeShell

	// Success case.
	fakeShell.commandOutput["sysupgrade -n some-file"] = "some output"
	TriggerFirmwareUpdate("some-file")
	assert.Contains(t, fakeShell.commandsRun, "sysupgrade -n some-file")

	// Error case.
	fakeShell.reset()
	fakeShell.commandErrors["sysupgrade -n some-file"] = errors.New("oops")
	TriggerFirmwareUpdate("some-file")
	assert.Contains(t, fakeShell.commandsRun, "sysupgrade -n some-file")
}

func TestRadio_determineAndSetVersion(t *testing.T) {
	fakeTree := newFakeUciTree()
	uciTree = fakeTree
	fakeShell := newFakeShell(t)
	shell = fakeShell

	// Vivid-Hosting success case.
	fakeTree.valuesForGet["system.@system[0].model"] = "VH-109(AP)"
	fakeShell.commandOutput["cat /etc/config/vh_firmware"] = "\tVH version 1.2.3 \n"
	radio := Radio{}
	radio.determineAndSetVersion()
	assert.Equal(t, "VH version 1.2.3", radio.Version)

	// Vivid-Hosting error case.
	fakeTree.reset()
	fakeTree.valuesForGet["system.@system[0].model"] = "VH-109(AP)"
	fakeShell.reset()
	fakeShell.commandErrors["cat /etc/config/vh_firmware"] = errors.New("oops")
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
