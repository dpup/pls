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

func TestRun_ReturnsError(t *testing.T) {
	var stdout bytes.Buffer
	err := Run("false", &stdout)
	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestRun_HandlesPipes(t *testing.T) {
	var stdout bytes.Buffer
	err := Run("echo hello world | tr ' ' '\\n' | wc -l", &stdout)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}
