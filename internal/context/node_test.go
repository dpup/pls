package context

import (
	"path/filepath"
	"testing"
)

func TestNodeParser_WithPackageJSON(t *testing.T) {
	dir := t.TempDir()

	packageJSON := `{
  "name": "my-app",
  "scripts": {
    "test": "jest",
    "build": "tsc",
    "lint": "eslint ."
  },
  "workspaces": ["packages/*"]
}`
	writeFile(t, filepath.Join(dir, "package.json"), packageJSON)
	// bun.lockb indicates bun package manager
	writeFile(t, filepath.Join(dir, "bun.lockb"), "")

	p := &NodeParser{}
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
	if len(scripts) != 3 {
		t.Errorf("expected 3 scripts, got %d: %v", len(scripts), scripts)
	}

	pm, ok := result.Data["package_manager"].(string)
	if !ok {
		t.Fatalf("expected package_manager to be string, got %T", result.Data["package_manager"])
	}
	if pm != "bun" {
		t.Errorf("expected package_manager 'bun', got %q", pm)
	}

	workspaces, ok := result.Data["workspaces"].([]string)
	if !ok {
		t.Fatalf("expected workspaces to be []string, got %T", result.Data["workspaces"])
	}
	if len(workspaces) != 1 || workspaces[0] != "packages/*" {
		t.Errorf("expected workspaces [packages/*], got %v", workspaces)
	}
}

func TestNodeParser_NoPackageJSON(t *testing.T) {
	dir := t.TempDir()

	p := &NodeParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no package.json")
	}
}

func TestNodeParser_SubdirDetection(t *testing.T) {
	dir := t.TempDir()

	packageJSON := `{
  "name": "my-app",
  "scripts": {
    "dev": "vite"
  }
}`
	writeFile(t, filepath.Join(dir, "package.json"), packageJSON)

	sub := filepath.Join(dir, "src", "lib")
	writeFile(t, filepath.Join(sub, ".keep"), "")

	p := &NodeParser{}
	result, err := p.Parse(dir, sub)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result from subdir, got nil")
		return
	}

	scripts, ok := result.Data["scripts"].([]string)
	if !ok {
		t.Fatalf("expected scripts to be []string, got %T", result.Data["scripts"])
	}
	if len(scripts) != 1 {
		t.Errorf("expected 1 script, got %d: %v", len(scripts), scripts)
	}

	// Default package manager should be npm when no lockfile present
	pm := result.Data["package_manager"].(string)
	if pm != "npm" {
		t.Errorf("expected default package_manager 'npm', got %q", pm)
	}
}
