package radio

import "os/exec"

// shellWrapper is an interface to wrap running CLI commands, to facilitate testing.
type shellWrapper interface {
	// runCommand runs the given command with the given arguments and returns the output.
	runCommand(command string, args ...string) (string, error)

	// startCommand starts the given command with the given arguments without waiting for it to finish.
	startCommand(command string, args ...string) error
}

// execShell is an implementation of the shellWrapper interface that runs commands using the exec package.
type execShell struct{}

func (shell execShell) runCommand(command string, args ...string) (string, error) {
	outputBytes, err := exec.Command(command, args...).CombinedOutput()
	return string(outputBytes), err
}

func (shell execShell) startCommand(command string, args ...string) error {
	return exec.Command(command, args...).Start()
}
