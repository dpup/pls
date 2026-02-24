package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// maxToolTurns is the maximum number of tool-use round trips before forcing a
// final text response. Keeping this low bounds latency for the CLI.
const maxToolTurns = 2

// toolDefs returns the tool definitions offered to the LLM.
func toolDefs() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		{
			OfTool: &anthropic.ToolParam{
				Name:        "list_files",
				Description: anthropic.String("List files and subdirectories at a path relative to the repo root. Returns names with trailing '/' for directories."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Directory path relative to repo root, e.g. '.' or 'internal/history'",
						},
					},
					Required: []string{"path"},
				},
			},
		},
		{
			OfTool: &anthropic.ToolParam{
				Name:        "read_file",
				Description: anthropic.String("Read the contents of a file relative to the repo root. Returns up to 200 lines."),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "File path relative to repo root",
						},
						"max_lines": map[string]any{
							"type":        "integer",
							"description": "Maximum number of lines to return (default and max: 200)",
						},
					},
					Required: []string{"path"},
				},
			},
		},
	}
}

// toolHandler executes tools within a sandboxed repo root.
type toolHandler struct {
	repoRoot string
}

func (h *toolHandler) execute(name string, input json.RawMessage) (string, bool) {
	switch name {
	case "list_files":
		return h.listFiles(input)
	case "read_file":
		return h.readFile(input)
	default:
		return fmt.Sprintf("unknown tool: %s", name), true
	}
}

func (h *toolHandler) listFiles(input json.RawMessage) (string, bool) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}

	target := filepath.Join(h.repoRoot, filepath.FromSlash(params.Path))
	if !isWithin(target, h.repoRoot) {
		return "path is outside repo root", true
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return fmt.Sprintf("cannot read directory: %v", err), true
	}

	var lines []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			name += "/"
		}
		lines = append(lines, name)
	}
	return strings.Join(lines, "\n"), false
}

func (h *toolHandler) readFile(input json.RawMessage) (string, bool) {
	var params struct {
		Path     string `json:"path"`
		MaxLines int    `json:"max_lines"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	if params.MaxLines <= 0 || params.MaxLines > 200 {
		params.MaxLines = 200
	}

	target := filepath.Join(h.repoRoot, filepath.FromSlash(params.Path))
	if !isWithin(target, h.repoRoot) {
		return "path is outside repo root", true
	}

	// Check file size before reading to prevent OOM on large files.
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Sprintf("cannot read file: %v", err), true
	}
	if info.IsDir() {
		return fmt.Sprintf("%q is a directory, not a file. Use list_files to see its contents.", params.Path), true
	}
	const maxFileSize = 1 << 20 // 1MB
	if info.Size() > maxFileSize {
		return fmt.Sprintf("file too large (%d bytes, max %d). Try a more specific path.", info.Size(), maxFileSize), true
	}

	data, err := os.ReadFile(target)
	if err != nil {
		return fmt.Sprintf("cannot read file: %v", err), true
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > params.MaxLines {
		total := len(lines)
		lines = lines[:params.MaxLines]
		lines = append(lines, fmt.Sprintf("\n... truncated (%d lines total)", total))
	}
	return strings.Join(lines, "\n"), false
}

// isWithin checks that target is within the root directory (path traversal guard).
// Resolves symlinks to prevent escaping via symlinked paths.
func isWithin(target, root string) bool {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}

	// Resolve symlinks if the target exists on disk.
	if resolved, err := filepath.EvalSymlinks(absTarget); err == nil {
		absTarget = resolved
	}
	if resolved, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = resolved
	}

	return absTarget == absRoot || strings.HasPrefix(absTarget, absRoot+string(filepath.Separator))
}
