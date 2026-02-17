package exec

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Run executes a command string via the user's shell, streaming output to w.
// Uses $SHELL if set, falls back to /bin/sh.
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
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}
