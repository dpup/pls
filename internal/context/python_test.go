package context

import (
	"path/filepath"
	"testing"
)

func TestPythonParser_WithPyprojectAndPoetry(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[tool.poetry]\nname = \"myapp\"\n")
	writeFile(t, filepath.Join(dir, "poetry.lock"), "# lock")

	p := &PythonParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	pm, ok := result.Data["package_manager"].(string)
	if !ok {
		t.Fatalf("expected package_manager to be string, got %T", result.Data["package_manager"])
	}
	if pm != "poetry" {
		t.Errorf("expected package_manager 'poetry', got %q", pm)
	}

	hasPyproject, ok := result.Data["has_pyproject"].(bool)
	if !ok {
		t.Fatalf("expected has_pyproject to be bool, got %T", result.Data["has_pyproject"])
	}
	if !hasPyproject {
		t.Error("expected has_pyproject to be true")
	}
}

func TestPythonParser_WithUv(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\nname = \"myapp\"\n")
	writeFile(t, filepath.Join(dir, "uv.lock"), "# lock")

	p := &PythonParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	pm := result.Data["package_manager"].(string)
	if pm != "uv" {
		t.Errorf("expected package_manager 'uv', got %q", pm)
	}
}

func TestPythonParser_WithPipenv(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "Pipfile"), "[packages]\n")
	writeFile(t, filepath.Join(dir, "Pipfile.lock"), "{}")

	p := &PythonParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	pm := result.Data["package_manager"].(string)
	if pm != "pipenv" {
		t.Errorf("expected package_manager 'pipenv', got %q", pm)
	}
}

func TestPythonParser_WithRequirementsTxt(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "requirements.txt"), "flask==2.0\n")

	p := &PythonParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	pm := result.Data["package_manager"].(string)
	if pm != "pip" {
		t.Errorf("expected package_manager 'pip', got %q", pm)
	}

	hasPyproject, ok := result.Data["has_pyproject"].(bool)
	if !ok {
		t.Fatalf("expected has_pyproject to be bool, got %T", result.Data["has_pyproject"])
	}
	if hasPyproject {
		t.Error("expected has_pyproject to be false")
	}
}

func TestPythonParser_VenvDetection(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "pyproject.toml"), "[project]\n")
	writeFile(t, filepath.Join(dir, ".venv", ".keep"), "")

	p := &PythonParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
		return
	}

	venv, ok := result.Data["venv"].(string)
	if !ok {
		t.Fatalf("expected venv to be string, got %T", result.Data["venv"])
	}
	if venv != ".venv" {
		t.Errorf("expected venv '.venv', got %q", venv)
	}
}

func TestPythonParser_NoPythonProject(t *testing.T) {
	dir := t.TempDir()

	p := &PythonParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no Python indicators")
	}
}
