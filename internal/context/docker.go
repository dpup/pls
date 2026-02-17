package context

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var serviceNameRe = regexp.MustCompile(`^  ([a-zA-Z_][a-zA-Z0-9_-]*):\s*$`)

// DockerParser detects a Docker Compose file and extracts service names.
type DockerParser struct{}

func (d *DockerParser) Name() string { return "docker" }

func (d *DockerParser) Parse(repoRoot, cwd string) (*Result, error) {
	composeNames := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	var raw []byte
	var err error
	for _, name := range composeNames {
		raw, err = os.ReadFile(filepath.Join(repoRoot, name))
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if raw == nil {
		return nil, nil
	}

	// Simple line-based parsing: find "services:" header then indented service names
	var services []string
	inServices := false
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "services:" {
			inServices = true
			continue
		}
		if inServices {
			// If we hit a non-indented, non-empty line, we've left the services block
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
				break
			}
			if m := serviceNameRe.FindStringSubmatch(line); m != nil {
				services = append(services, m[1])
			}
		}
	}

	if len(services) == 0 {
		return nil, nil
	}

	return &Result{
		Name: d.Name(),
		Data: map[string]any{
			"services": services,
		},
	}, nil
}
