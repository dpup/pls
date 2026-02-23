package exec

import (
	"io"
	"os"
	"os/exec"
	"strconv"
)

// ExitError wraps a command's non-zero exit code.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return "exit status " + strconv.Itoa(e.Code)
}

// Run executes a command string via the user's shell, streaming output to w.
// Uses $SHELL if set, falls back to /bin/sh.
// Returns *ExitError if the command exits with a non-zero status.
func Run(command string, w io.Writer) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-c", command)
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ExitError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}
