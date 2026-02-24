package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_NoArgs(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when running with no args, got nil")
	}
}

func TestRun_MissingAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("HOME", t.TempDir()) // prevent loading api_key from user's config file
	cmd := newRootCmd()
	cmd.SetArgs([]string{"test intent"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error about missing API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key") && !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("expected error about API key, got: %v", err)
	}
}

func TestRun_ExplainFlag(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--explain", "run tests"})
	buf := new(bytes.Buffer)
	cmd.SetErr(buf)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--explain should not return an error, got: %v", err)
	}
}

func TestRun_VersionFlag(t *testing.T) {
	cmd := newRootCmd()
	cmd.Version = "test-version"
	cmd.SetArgs([]string{"--version"})
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--version should not error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "test-version") {
		t.Errorf("expected output to contain 'test-version', got: %q", buf.String())
	}
}
