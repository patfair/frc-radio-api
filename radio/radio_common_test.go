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
