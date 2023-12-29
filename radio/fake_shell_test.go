package radio

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// fakeShell stubs the shellWrapper interface for testing purposes.
type fakeShell struct {
	// Test assertion object, used to fail the test on unexpected commands.
	t *testing.T

	// Set of commands that have been run, for tests to assert against.
	commandsRun map[string]struct{}

	// Map of commands to their successful response, for tests to set. A given command should only appear once between
	// commandOutput and commandErrors.
	commandOutput map[string]string

	// Map of commands to their error response, for tests to set. A given command should only appear once between
	// commandOutput and commandErrors.
	commandErrors map[string]error
}

func newFakeShell(t *testing.T) *fakeShell {
	return &fakeShell{
		t:             t,
		commandsRun:   make(map[string]struct{}),
		commandOutput: make(map[string]string),
		commandErrors: make(map[string]error),
	}
}

func (shell *fakeShell) runCommand(command string, args ...string) (string, error) {
	fullCommand := strings.Join(append([]string{command}, args...), " ")
	shell.commandsRun[fullCommand] = struct{}{}
	if output, ok := shell.commandOutput[fullCommand]; ok {
		return output, nil
	}
	if err, ok := shell.commandErrors[fullCommand]; ok {
		return "", err
	}
	assert.Fail(shell.t, "unexpected command: "+fullCommand)
	return "", nil
}

func (shell *fakeShell) startCommand(command string, args ...string) error {
	fullCommand := strings.Join(append([]string{command}, args...), " ")
	shell.commandsRun[fullCommand] = struct{}{}
	if _, ok := shell.commandOutput[fullCommand]; ok {
		return nil
	}
	if err, ok := shell.commandErrors[fullCommand]; ok {
		return err
	}
	assert.Fail(shell.t, "unexpected command: "+fullCommand)
	return nil
}

// reset clears the state of the fake shell.
func (shell *fakeShell) reset() {
	shell.commandsRun = make(map[string]struct{})
	shell.commandOutput = make(map[string]string)
	shell.commandErrors = make(map[string]error)
}
