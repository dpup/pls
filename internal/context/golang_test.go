package context

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a test helper that writes a file and fails on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestGoParser_WithGoMod(t *testing.T) {
	dir := t.TempDir()

	goMod := `module github.com/example/myproject

go 1.21

require (
	github.com/spf13/cobra v1.8.0
)
`
	writeFile(t, filepath.Join(dir, "go.mod"), goMod)
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	writeFile(t, filepath.Join(dir, "main_test.go"), "package main")

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
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

	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/example/subtest\n\ngo 1.21\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main")

	sub := filepath.Join(dir, "internal", "pkg")
	writeFile(t, filepath.Join(sub, "util.go"), "package pkg")
	writeFile(t, filepath.Join(sub, "util_test.go"), "package pkg")

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
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

	writeFile(t, filepath.Join(dir, "go.mod"), "module github.com/example/notests\n\ngo 1.21\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main")

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
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

	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/m\n\ngo 1.21\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main")

	vendor := filepath.Join(dir, "vendor", "somepkg")
	writeFile(t, filepath.Join(vendor, "lib.go"), "package somepkg")
	writeFile(t, filepath.Join(vendor, "lib_test.go"), "package somepkg")

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
