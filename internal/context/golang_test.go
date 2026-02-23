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

	testPkgs, ok := result.Data["test_packages"].([]string)
	if !ok {
		t.Fatalf("expected test_packages to be []string, got %T", result.Data["test_packages"])
	}
	if len(testPkgs) == 0 {
		t.Error("expected at least one test package")
	}
	if testPkgs[0] != "." {
		t.Errorf("expected root test package '.', got %q", testPkgs[0])
	}
}

func TestGoParser_WithGoModSubdirTests(t *testing.T) {
	dir := t.TempDir()

	goMod := `module github.com/example/subtest

go 1.21
`
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	// Create a test file in a nested subdir.
	sub := filepath.Join(dir, "internal", "pkg")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "util.go"), []byte("package pkg"), 0o644)
	os.WriteFile(filepath.Join(sub, "util_test.go"), []byte("package pkg"), 0o644)

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	testPkgs, ok := result.Data["test_packages"].([]string)
	if !ok {
		t.Fatalf("expected test_packages to be []string, got %T", result.Data["test_packages"])
	}
	found := false
	for _, p := range testPkgs {
		if p == "./internal/pkg" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected test_packages to include './internal/pkg', got %v", testPkgs)
	}

	// Check packages list includes both root and subdir.
	pkgs, ok := result.Data["packages"].([]string)
	if !ok {
		t.Fatalf("expected packages to be []string, got %T", result.Data["packages"])
	}
	if len(pkgs) < 2 {
		t.Errorf("expected at least 2 packages (root + subdir), got %v", pkgs)
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

	if result.Data["test_packages"] != nil {
		t.Errorf("expected no test_packages, got %v", result.Data["test_packages"])
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

func TestGoParser_SkipsVendor(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n\ngo 1.21\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)

	vendor := filepath.Join(dir, "vendor", "somepkg")
	os.MkdirAll(vendor, 0o755)
	os.WriteFile(filepath.Join(vendor, "lib.go"), []byte("package somepkg"), 0o644)
	os.WriteFile(filepath.Join(vendor, "lib_test.go"), []byte("package somepkg"), 0o644)

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	pkgs, _ := result.Data["packages"].([]string)
	for _, pkg := range pkgs {
		if pkg == "./vendor/somepkg" {
			t.Error("packages should not include vendor directory")
		}
	}
}
