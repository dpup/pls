package context

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestScriptsParser_WithBinDir(t *testing.T) {
	dir := t.TempDir()

	binDir := filepath.Join(dir, "bin")
	writeFile(t, filepath.Join(binDir, "deploy.sh"), "#!/bin/bash\n")
	writeFile(t, filepath.Join(binDir, "setup"), "#!/bin/bash\n")

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	scripts, ok := result.Data["scripts"].([]string)
	if !ok {
		t.Fatalf("expected scripts to be []string, got %T", result.Data["scripts"])
	}

	sort.Strings(scripts)
	expected := []string{"bin/deploy.sh", "bin/setup"}
	if len(scripts) != len(expected) {
		t.Fatalf("expected %d scripts, got %d: %v", len(expected), len(scripts), scripts)
	}
	for i, e := range expected {
		if scripts[i] != e {
			t.Errorf("script[%d]: expected %q, got %q", i, e, scripts[i])
		}
	}
}

func TestScriptsParser_WithScriptsDir(t *testing.T) {
	dir := t.TempDir()

	scriptsDir := filepath.Join(dir, "scripts")
	writeFile(t, filepath.Join(scriptsDir, "migrate.py"), "#!/usr/bin/env python\n")

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	scripts, ok := result.Data["scripts"].([]string)
	if !ok {
		t.Fatalf("expected scripts to be []string, got %T", result.Data["scripts"])
	}
	if len(scripts) != 1 || scripts[0] != "scripts/migrate.py" {
		t.Errorf("expected [scripts/migrate.py], got %v", scripts)
	}
}

func TestScriptsParser_BothDirs(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "bin", "run.sh"), "#!/bin/bash\n")
	writeFile(t, filepath.Join(dir, "scripts", "test.sh"), "#!/bin/bash\n")

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	scripts, ok := result.Data["scripts"].([]string)
	if !ok {
		t.Fatalf("expected scripts to be []string, got %T", result.Data["scripts"])
	}

	sort.Strings(scripts)
	expected := []string{"bin/run.sh", "scripts/test.sh"}
	if len(scripts) != len(expected) {
		t.Fatalf("expected %d scripts, got %d: %v", len(expected), len(scripts), scripts)
	}
	for i, e := range expected {
		if scripts[i] != e {
			t.Errorf("script[%d]: expected %q, got %q", i, e, scripts[i])
		}
	}
}

func TestScriptsParser_SkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()

	binDir := filepath.Join(dir, "bin")
	writeFile(t, filepath.Join(binDir, "deploy.sh"), "#!/bin/bash\n")
	// Create a file in a subdirectory inside bin/ — should be skipped
	writeFile(t, filepath.Join(binDir, "helpers", "util.sh"), "#!/bin/bash\n")

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	scripts, ok := result.Data["scripts"].([]string)
	if !ok {
		t.Fatalf("expected scripts to be []string, got %T", result.Data["scripts"])
	}
	if len(scripts) != 1 || scripts[0] != "bin/deploy.sh" {
		t.Errorf("expected [bin/deploy.sh], got %v", scripts)
	}
}

func TestScriptsParser_NoScriptDirs(t *testing.T) {
	dir := t.TempDir()

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no script directories")
	}
}

func TestScriptsParser_EmptyDirs(t *testing.T) {
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when script dirs are empty")
	}
}
