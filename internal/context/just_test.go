package context

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestJustParser_WithJustfile(t *testing.T) {
	dir := t.TempDir()

	justfile := `# Justfile for project

test *args:
    cargo test {{args}}

build:
    cargo build --release
`
	os.WriteFile(filepath.Join(dir, "Justfile"), []byte(justfile), 0o644)

	p := &JustParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	recipes, ok := result.Data["recipes"].([]string)
	if !ok {
		t.Fatalf("expected recipes to be []string, got %T", result.Data["recipes"])
	}

	sort.Strings(recipes)
	expected := []string{"build", "test"}
	if len(recipes) != len(expected) {
		t.Fatalf("expected %d recipes, got %d: %v", len(expected), len(recipes), recipes)
	}
	for i, e := range expected {
		if recipes[i] != e {
			t.Errorf("recipe[%d]: expected %q, got %q", i, e, recipes[i])
		}
	}
}

func TestJustParser_NoJustfile(t *testing.T) {
	dir := t.TempDir()

	p := &JustParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no Justfile")
	}
}

func TestJustParser_LowercaseJustfile(t *testing.T) {
	dir := t.TempDir()

	justfile := `deploy:
    kubectl apply -f deploy.yaml
`
	os.WriteFile(filepath.Join(dir, "justfile"), []byte(justfile), 0o644)

	p := &JustParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	recipes, ok := result.Data["recipes"].([]string)
	if !ok {
		t.Fatalf("expected recipes to be []string, got %T", result.Data["recipes"])
	}
	if len(recipes) != 1 || recipes[0] != "deploy" {
		t.Errorf("expected [deploy], got %v", recipes)
	}
}
