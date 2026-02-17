package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoParser_WithGoMod(t *testing.T) {
	dir := t.TempDir()

	goMod := `module github.com/example/myproject

go 1.21

require (
	github.com/spf13/cobra v1.8.0
)
`
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main"), 0o644)

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	module, ok := result.Data["module"].(string)
	if !ok {
		t.Fatalf("expected module to be string, got %T", result.Data["module"])
	}
	if module != "github.com/example/myproject" {
		t.Errorf("expected module 'github.com/example/myproject', got %q", module)
	}

	hasTests, ok := result.Data["has_tests"].(bool)
	if !ok {
		t.Fatalf("expected has_tests to be bool, got %T", result.Data["has_tests"])
	}
	if !hasTests {
		t.Error("expected has_tests to be true")
	}
}

func TestGoParser_WithGoModSubdirTests(t *testing.T) {
	dir := t.TempDir()

	goMod := `module github.com/example/subtest

go 1.21
`
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	// Create a test file one level deep in a subdir
	sub := filepath.Join(dir, "pkg")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "util_test.go"), []byte("package pkg"), 0o644)

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	hasTests, ok := result.Data["has_tests"].(bool)
	if !ok {
		t.Fatalf("expected has_tests to be bool, got %T", result.Data["has_tests"])
	}
	if !hasTests {
		t.Error("expected has_tests to be true when test file in subdir")
	}
}

func TestGoParser_NoTests(t *testing.T) {
	dir := t.TempDir()

	goMod := `module github.com/example/notests

go 1.21
`
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	hasTests, ok := result.Data["has_tests"].(bool)
	if !ok {
		t.Fatalf("expected has_tests to be bool, got %T", result.Data["has_tests"])
	}
	if hasTests {
		t.Error("expected has_tests to be false when no test files")
	}
}

func TestGoParser_NoGoMod(t *testing.T) {
	dir := t.TempDir()

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no go.mod")
	}
}
