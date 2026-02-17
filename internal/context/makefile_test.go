package context

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestMakeParser_WithMakefile(t *testing.T) {
	dir := t.TempDir()

	makefile := `.PHONY: test lint build

test:
	go test ./...

lint:
	golangci-lint run

build:
	go build -o bin/app .

# This is a comment
	indented-not-a-target:
`
	os.WriteFile(filepath.Join(dir, "Makefile"), []byte(makefile), 0o644)

	p := &MakeParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	targets, ok := result.Data["targets"].([]string)
	if !ok {
		t.Fatalf("expected targets to be []string, got %T", result.Data["targets"])
	}

	sort.Strings(targets)
	expected := []string{"build", "lint", "test"}
	if len(targets) != len(expected) {
		t.Fatalf("expected %d targets, got %d: %v", len(expected), len(targets), targets)
	}
	for i, e := range expected {
		if targets[i] != e {
			t.Errorf("target[%d]: expected %q, got %q", i, e, targets[i])
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
