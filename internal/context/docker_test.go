package context

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDockerParser_WithDockerCompose(t *testing.T) {
	dir := t.TempDir()

	compose := `version: "3.8"

services:
  web:
    build: .
    ports:
      - "8080:8080"
  db:
    image: postgres:15
    environment:
      POSTGRES_DB: mydb
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0o644)

	p := &DockerParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	services, ok := result.Data["services"].([]string)
	if !ok {
		t.Fatalf("expected services to be []string, got %T", result.Data["services"])
	}

	sort.Strings(services)
	expected := []string{"db", "web"}
	if len(services) != len(expected) {
		t.Fatalf("expected %d services, got %d: %v", len(expected), len(services), services)
	}
	for i, e := range expected {
		if services[i] != e {
			t.Errorf("service[%d]: expected %q, got %q", i, e, services[i])
		}
	}
}

func TestDockerParser_ComposeYaml(t *testing.T) {
	dir := t.TempDir()

	compose := `services:
  api:
    build: .
`
	os.WriteFile(filepath.Join(dir, "compose.yaml"), []byte(compose), 0o644)

	p := &DockerParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}

	services, ok := result.Data["services"].([]string)
	if !ok {
		t.Fatalf("expected services to be []string, got %T", result.Data["services"])
	}
	if len(services) != 1 || services[0] != "api" {
		t.Errorf("expected [api], got %v", services)
	}
}

func TestDockerParser_NoComposeFile(t *testing.T) {
	dir := t.TempDir()

	p := &DockerParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no compose file")
	}
}

func TestDockerParser_NoServices(t *testing.T) {
	dir := t.TempDir()

	compose := `version: "3.8"
# no services defined
`
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0o644)

	p := &DockerParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result when no services")
	}
}
