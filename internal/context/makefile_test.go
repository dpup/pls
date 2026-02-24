package context

import (
	"path/filepath"
	"testing"
)

func TestMakeParser_WithMakefile(t *testing.T) {
	dir := t.TempDir()

	makefile := `.PHONY: test lint build

test: ## Run all tests
	go test ./...

lint:
	golangci-lint run

build: ## Build the binary
	go build -o bin/app .

# This is a comment
	indented-not-a-target:
`
	writeFile(t, filepath.Join(dir, "Makefile"), makefile)

	p := &MakeParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	targets, ok := result.Data["targets"].(map[string]string)
	if !ok {
		t.Fatalf("expected targets to be map[string]string, got %T", result.Data["targets"])
	}

	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d: %v", len(targets), targets)
	}

	expected := map[string]string{
		"test":  "Run all tests",
		"lint":  "",
		"build": "Build the binary",
	}
	for name, desc := range expected {
		got, exists := targets[name]
		if !exists {
			t.Errorf("missing target %q", name)
		} else if got != desc {
			t.Errorf("target %q: expected desc %q, got %q", name, desc, got)
		}
	}
}

func TestMakeParser_NoMakefile(t *testing.T) {
	dir := t.TempDir()

	p := &MakeParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no Makefile")
	}
}
