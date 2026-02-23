package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun_NoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when running with no args, got nil")
	}
}

func TestRun_MissingAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	rootCmd.SetArgs([]string{"test intent"})
	defer rootCmd.SetArgs([]string{})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error about missing API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key") && !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Errorf("expected error about API key, got: %v", err)
	}
}

func TestRun_ExplainFlag(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	rootCmd.SetArgs([]string{"--explain", "run tests"})
	defer func() {
		rootCmd.SetArgs([]string{})
		explain = false
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetErr(buf)
	defer rootCmd.SetErr(nil)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--explain should not return an error, got: %v", err)
	}
}

func TestRun_VersionFlag(t *testing.T) {
	rootCmd.Version = "test-version"
	rootCmd.SetArgs([]string{"--version"})
	defer rootCmd.SetArgs([]string{})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	defer rootCmd.SetOut(nil)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--version should not error, got: %v", err)
	}
	if !strings.Contains(buf.String(), "test-version") {
		t.Errorf("expected output to contain 'test-version', got: %q", buf.String())
	}
}
