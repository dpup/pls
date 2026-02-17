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
	os.MkdirAll(binDir, 0o755)
	os.WriteFile(filepath.Join(binDir, "deploy.sh"), []byte("#!/bin/bash\n"), 0o755)
	os.WriteFile(filepath.Join(binDir, "setup"), []byte("#!/bin/bash\n"), 0o755)

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
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
	os.MkdirAll(scriptsDir, 0o755)
	os.WriteFile(filepath.Join(scriptsDir, "migrate.py"), []byte("#!/usr/bin/env python\n"), 0o755)

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
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

	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)
	os.WriteFile(filepath.Join(binDir, "run.sh"), []byte("#!/bin/bash\n"), 0o755)

	scriptsDir := filepath.Join(dir, "scripts")
	os.MkdirAll(scriptsDir, 0o755)
	os.WriteFile(filepath.Join(scriptsDir, "test.sh"), []byte("#!/bin/bash\n"), 0o755)

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
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
	os.MkdirAll(binDir, 0o755)
	os.WriteFile(filepath.Join(binDir, "deploy.sh"), []byte("#!/bin/bash\n"), 0o755)
	// Create a subdirectory inside bin/ — should be skipped
	os.MkdirAll(filepath.Join(binDir, "helpers"), 0o755)
	os.WriteFile(filepath.Join(binDir, "helpers", "util.sh"), []byte("#!/bin/bash\n"), 0o755)

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
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

	os.MkdirAll(filepath.Join(dir, "bin"), 0o755)
	os.MkdirAll(filepath.Join(dir, "scripts"), 0o755)

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when script dirs are empty")
	}
}
