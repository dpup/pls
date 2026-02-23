package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// extractJSON tests
// ---------------------------------------------------------------------------

func TestExtractJSON_CleanJSON(t *testing.T) {
	input := `{"candidates":[]}`
	got := extractJSON(input)
	if got != input {
		t.Errorf("expected %q, got %q", input, got)
	}
}

func TestExtractJSON_MarkdownFences(t *testing.T) {
	input := "```json\n{\"candidates\":[{\"cmd\":\"echo hi\"}]}\n```"
	want := `{"candidates":[{"cmd":"echo hi"}]}`
	got := extractJSON(input)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestExtractJSON_PreambleText(t *testing.T) {
	inner := `{"candidates":[{"cmd":"ls -la","confidence":0.9}]}`
	input := "Here are the results:\n" + inner
	got := extractJSON(input)
	if got != inner {
		t.Errorf("expected %q, got %q", inner, got)
	}
}

func TestExtractJSON_NoJSON(t *testing.T) {
	input := "no json here at all"
	got := extractJSON(input)
	if got != input {
		t.Errorf("expected input unchanged %q, got %q", input, got)
	}
}

func TestExtractJSON_TextAfterJSON(t *testing.T) {
	inner := `{"candidates":[{"cmd":"pwd"}]}`
	input := "Here: " + inner + "\nHope that helps"
	got := extractJSON(input)
	if got != inner {
		t.Errorf("expected %q, got %q", inner, got)
	}
}

// ---------------------------------------------------------------------------
// toolHandler tests — helpers
// ---------------------------------------------------------------------------

// newTestHandler creates a toolHandler rooted at a temp directory and returns
// the handler plus a cleanup function.
func newTestHandler(t *testing.T) *toolHandler {
	t.Helper()
	dir := t.TempDir()
	return &toolHandler{repoRoot: dir}
}

// writeTestFile creates a file under the handler's repoRoot.
func writeTestFile(t *testing.T, h *toolHandler, relPath, content string) {
	t.Helper()
	full := filepath.Join(h.repoRoot, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

// makeTestDir creates a subdirectory under the handler's repoRoot.
func makeTestDir(t *testing.T, h *toolHandler, relPath string) {
	t.Helper()
	full := filepath.Join(h.repoRoot, filepath.FromSlash(relPath))
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}

// ---------------------------------------------------------------------------
// toolHandler: list_files
// ---------------------------------------------------------------------------

func TestToolHandler_ListFiles(t *testing.T) {
	h := newTestHandler(t)

	writeTestFile(t, h, "alpha.txt", "a")
	writeTestFile(t, h, "beta.go", "b")
	makeTestDir(t, h, "subdir")

	input := json.RawMessage(`{"path": "."}`)
	result, isErr := h.execute("list_files", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	lines := strings.Split(result, "\n")
	got := map[string]bool{}
	for _, l := range lines {
		got[l] = true
	}

	if !got["alpha.txt"] {
		t.Errorf("expected alpha.txt in output, got: %s", result)
	}
	if !got["beta.go"] {
		t.Errorf("expected beta.go in output, got: %s", result)
	}
	if !got["subdir/"] {
		t.Errorf("expected subdir/ in output (trailing slash for dirs), got: %s", result)
	}
}

func TestToolHandler_ListFiles_HiddenFilesSkipped(t *testing.T) {
	h := newTestHandler(t)

	writeTestFile(t, h, ".hidden", "secret")
	writeTestFile(t, h, "visible.txt", "hello")
	makeTestDir(t, h, ".git")

	input := json.RawMessage(`{"path": "."}`)
	result, isErr := h.execute("list_files", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	if strings.Contains(result, ".hidden") {
		t.Errorf("hidden files should be skipped, got: %s", result)
	}
	if strings.Contains(result, ".git") {
		t.Errorf("hidden directories should be skipped, got: %s", result)
	}
	if !strings.Contains(result, "visible.txt") {
		t.Errorf("visible files should be present, got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// toolHandler: read_file
// ---------------------------------------------------------------------------

func TestToolHandler_ReadFile(t *testing.T) {
	h := newTestHandler(t)
	writeTestFile(t, h, "hello.txt", "line1\nline2\nline3\n")

	input := json.RawMessage(`{"path": "hello.txt"}`)
	result, isErr := h.execute("read_file", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	if !strings.Contains(result, "line1") {
		t.Errorf("result should contain file content, got: %s", result)
	}
	if !strings.Contains(result, "line3") {
		t.Errorf("result should contain file content, got: %s", result)
	}
}

func TestToolHandler_ReadFile_MaxLines(t *testing.T) {
	h := newTestHandler(t)

	// Build a file with 300 lines.
	var lines []string
	for i := 1; i <= 300; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	writeTestFile(t, h, "big.txt", strings.Join(lines, "\n"))

	input := json.RawMessage(`{"path": "big.txt"}`)
	result, isErr := h.execute("read_file", input)
	if isErr {
		t.Fatalf("unexpected error: %s", result)
	}

	if !strings.Contains(result, "line 1") {
		t.Errorf("should contain the first line, got: %s", result[:80])
	}
	if !strings.Contains(result, "line 200") {
		t.Errorf("should contain line 200, got last 80 chars: %s", result[len(result)-80:])
	}
	if strings.Contains(result, "line 201\n") {
		t.Errorf("should NOT contain line 201 (only 200 returned)")
	}
	if !strings.Contains(result, "truncated") {
		t.Errorf("should contain truncation message, got last 80 chars: %s", result[len(result)-80:])
	}
}

func TestToolHandler_ReadFile_Directory(t *testing.T) {
	h := newTestHandler(t)
	makeTestDir(t, h, "mydir")

	input := json.RawMessage(`{"path": "mydir"}`)
	result, isErr := h.execute("read_file", input)
	if !isErr {
		t.Fatalf("expected isError=true when reading a directory, got false")
	}
	if !strings.Contains(result, "directory") {
		t.Errorf("should hint that the path is a directory, got: %s", result)
	}
	if !strings.Contains(result, "list_files") {
		t.Errorf("should suggest list_files, got: %s", result)
	}
}

// ---------------------------------------------------------------------------
// toolHandler: path traversal
// ---------------------------------------------------------------------------

func TestToolHandler_PathTraversal(t *testing.T) {
	h := newTestHandler(t)

	// Attempt to read outside the repo root.
	input := json.RawMessage(`{"path": "../../../etc/passwd"}`)
	result, isErr := h.execute("read_file", input)
	if !isErr {
		t.Fatalf("expected isError=true for path traversal, got false")
	}
	if !strings.Contains(result, "outside") {
		t.Errorf("should mention path is outside repo root, got: %s", result)
	}

	// Also verify list_files rejects traversal.
	result2, isErr2 := h.execute("list_files", input)
	if !isErr2 {
		t.Fatalf("expected isError=true for path traversal on list_files, got false")
	}
	if !strings.Contains(result2, "outside") {
		t.Errorf("should mention path is outside repo root, got: %s", result2)
	}
}

// ---------------------------------------------------------------------------
// isWithin
// ---------------------------------------------------------------------------

func TestIsWithin(t *testing.T) {
	tests := []struct {
		name   string
		target string
		root   string
		want   bool
	}{
		{
			name:   "exact match",
			target: "/home/user/project",
			root:   "/home/user/project",
			want:   true,
		},
		{
			name:   "child file",
			target: "/home/user/project/foo.txt",
			root:   "/home/user/project",
			want:   true,
		},
		{
			name:   "nested child",
			target: "/home/user/project/a/b/c.txt",
			root:   "/home/user/project",
			want:   true,
		},
		{
			name:   "parent directory",
			target: "/home/user",
			root:   "/home/user/project",
			want:   false,
		},
		{
			name:   "sibling directory",
			target: "/home/user/other",
			root:   "/home/user/project",
			want:   false,
		},
		{
			name:   "prefix trick (projectX should not match project)",
			target: "/home/user/projectX/secret",
			root:   "/home/user/project",
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isWithin(tc.target, tc.root)
			if got != tc.want {
				t.Errorf("isWithin(%q, %q) = %v, want %v", tc.target, tc.root, got, tc.want)
			}
		})
	}
}
