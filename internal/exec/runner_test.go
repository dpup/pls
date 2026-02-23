package exec

import (
	"bytes"
	"testing"
)

func TestRun_CapturesOutput(t *testing.T) {
	var stdout bytes.Buffer
	err := Run("echo hello", &stdout)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stdout.String() != "hello\n" {
		t.Errorf("unexpected output: %q", stdout.String())
	}
}

func TestRun_ReturnsExitError(t *testing.T) {
	var stdout bytes.Buffer
	err := Run("exit 42", &stdout)
	if err == nil {
		t.Fatal("expected error for failing command")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 42 {
		t.Errorf("expected exit code 42, got %d", exitErr.Code)
	}
}

func TestRun_HandlesPipes(t *testing.T) {
	var stdout bytes.Buffer
	err := Run("echo hello world | tr ' ' '\\n' | wc -l", &stdout)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}
