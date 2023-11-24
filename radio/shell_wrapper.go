package radio

import "os/exec"

// shellWrapper is an interface to wrap running CLI commands, to facilitate testing.
type shellWrapper interface {
	// runCommand runs the given command with the given arguments and returns the output.
	runCommand(command string, args ...string) (string, error)
}

// execShell is an implementation of the shellWrapper interface that runs commands using the exec package.
type execShell struct{}

func (shell execShell) runCommand(command string, args ...string) (string, error) {
	outputBytes, err := exec.Command(command, args...).Output()
	return string(outputBytes), err
}
