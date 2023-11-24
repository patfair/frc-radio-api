package radio

import (
	"strings"
	"testing"
)

// fakeShell stubs the shellWrapper interface for testing purposes.
type fakeShell struct {
	t             *testing.T
	commandsRun   map[string]struct{}
	commandOutput map[string]string
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
	shell.t.Error("unexpected command: " + fullCommand)
	return "", nil
}

// reset clears the state of the fake shell.
func (shell *fakeShell) reset() {
	shell.commandsRun = make(map[string]struct{})
	shell.commandOutput = make(map[string]string)
	shell.commandErrors = make(map[string]error)
}
