# pls Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a CLI tool that translates natural language into project-aware shell commands using Claude, with history-based learning and a styled interactive TUI.

**Architecture:** Three layers — context parsers detect project tooling, LLM layer generates candidates using context + history, bubbletea TUI presents results with risk coloring. SQLite stores command history. No scoring engine — the LLM reasons over raw history.

**Tech Stack:** Go 1.25, Cobra (CLI), modernc.org/sqlite (pure-Go SQLite), github.com/anthropics/anthropic-sdk-go (Claude API), bubbletea (TUI), lipgloss (styling), BurntSushi/toml (config)

---

## Package Layout

```
pls/
├── main.go                       # Entry point
├── cmd/
│   └── root.go                   # Cobra root command, wires everything
├── internal/
│   ├── context/
│   │   ├── parser.go             # Parser interface + Context type
│   │   ├── collect.go            # Runs all parsers, builds context
│   │   ├── git.go                # Git parser
│   │   ├── node.go               # Node/package.json parser
│   │   ├── makefile.go           # Makefile parser
│   │   ├── just.go               # Justfile parser
│   │   ├── golang.go             # Go module parser
│   │   ├── docker.go             # docker-compose parser
│   │   ├── python.go             # Python project parser
│   │   └── scripts.go            # bin/ and scripts/ parser
│   ├── history/
│   │   ├── store.go              # SQLite store (open, migrate, CRUD)
│   │   └── store_test.go         # Store tests
│   ├── llm/
│   │   ├── client.go             # Anthropic client wrapper
│   │   ├── prompt.go             # Prompt building from context + history
│   │   └── types.go              # Candidate, Response types
│   ├── tui/
│   │   ├── model.go              # bubbletea Model (Init, Update, View)
│   │   └── styles.go             # lipgloss styles + risk colors
│   ├── exec/
│   │   └── runner.go             # Command execution + output streaming
│   └── config/
│       └── config.go             # TOML config loading
├── go.mod
└── go.sum
```

---

### Task 1: Project Scaffold + Cobra CLI

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`

**Step 1: Initialize Go module and install Cobra**

Run:
```bash
cd /workspace
go mod init github.com/dpup/pls
go get github.com/spf13/cobra@latest
```

**Step 2: Create main.go**

```go
package main

import "github.com/dpup/pls/cmd"

func main() {
	cmd.Execute()
}
```

**Step 3: Create cmd/root.go**

The root command takes all remaining args as the intent string.

```go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pls [intent]",
	Short: "Project-aware natural language shell command router",
	Long:  "Translates natural language into the right shell command for your current project.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		intent := strings.Join(args, " ")
		fmt.Printf("Intent: %s\n", intent)
		// TODO: wire up context -> llm -> tui pipeline
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

**Step 4: Verify it builds and runs**

Run:
```bash
go mod tidy
go build -o pls .
./pls "hello world"
```
Expected: `Intent: hello world`

**Step 5: Commit**

```bash
git add go.mod go.sum main.go cmd/
git commit -m "feat: scaffold project with Cobra CLI"
```

---

### Task 2: History Store (SQLite)

**Files:**
- Create: `internal/history/store.go`
- Create: `internal/history/store_test.go`

**Step 1: Install SQLite dependency**

Run:
```bash
go get modernc.org/sqlite@latest
```

**Step 2: Write failing tests for the store**

Create `internal/history/store_test.go`:

```go
package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestOpenCreatesDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
}

func TestRecordAndQueryProjectHistory(t *testing.T) {
	store := openTestStore(t)

	repoID, err := store.EnsureRepo("/home/user/myproject")
	if err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}

	err = store.Record(repoID, "src", "run tests", "go test ./...", OutcomeAccepted)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	err = store.Record(repoID, "src", "run linter", "golangci-lint run", OutcomeAccepted)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	entries, err := store.ProjectHistory(repoID, "src", 20)
	if err != nil {
		t.Fatalf("ProjectHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Command != "golangci-lint run" {
		t.Errorf("expected most recent first, got %q", entries[0].Command)
	}
}

func TestRecentGlobalHistory(t *testing.T) {
	store := openTestStore(t)

	repo1, _ := store.EnsureRepo("/project1")
	repo2, _ := store.EnsureRepo("/project2")

	store.Record(repo1, ".", "build", "make build", OutcomeAccepted)
	store.Record(repo2, ".", "search", "rg pattern", OutcomeAccepted)
	store.Record(repo1, ".", "test", "go test", OutcomeRejected)

	entries, err := store.RecentGlobal(10)
	if err != nil {
		t.Fatalf("RecentGlobal: %v", err)
	}
	// Only accepted commands in global history
	if len(entries) != 2 {
		t.Fatalf("expected 2 accepted entries, got %d", len(entries))
	}
}

func TestEnsureRepoIsIdempotent(t *testing.T) {
	store := openTestStore(t)

	id1, _ := store.EnsureRepo("/same/path")
	id2, _ := store.EnsureRepo("/same/path")
	if id1 != id2 {
		t.Errorf("expected same id, got %d and %d", id1, id2)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}
```

**Step 3: Run tests to verify they fail**

Run: `go test ./internal/history/ -v`
Expected: compilation errors (types don't exist yet)

**Step 4: Implement the store**

Create `internal/history/store.go`:

```go
package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	OutcomeAccepted = "accepted"
	OutcomeRejected = "rejected"
	OutcomeCopied   = "copied"
)

type Entry struct {
	ID        int64
	RepoID    int64
	CwdRel    string
	Intent    string
	Command   string
	Outcome   string
	CreatedAt time.Time
}

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS repos (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			root_path  TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS history (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id    INTEGER REFERENCES repos(id),
			cwd_rel    TEXT NOT NULL,
			intent     TEXT NOT NULL,
			command    TEXT NOT NULL,
			outcome    TEXT NOT NULL CHECK (outcome IN ('accepted', 'rejected', 'copied')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_history_repo_cwd ON history(repo_id, cwd_rel);
		CREATE INDEX IF NOT EXISTS idx_history_created ON history(created_at);
	`)
	return err
}

func (s *Store) EnsureRepo(rootPath string) (int64, error) {
	res, err := s.db.Exec(
		"INSERT INTO repos (root_path) VALUES (?) ON CONFLICT (root_path) DO NOTHING",
		rootPath,
	)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil || id == 0 {
		row := s.db.QueryRow("SELECT id FROM repos WHERE root_path = ?", rootPath)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (s *Store) Record(repoID int64, cwdRel, intent, command, outcome string) error {
	_, err := s.db.Exec(
		"INSERT INTO history (repo_id, cwd_rel, intent, command, outcome) VALUES (?, ?, ?, ?, ?)",
		repoID, cwdRel, intent, command, outcome,
	)
	return err
}

func (s *Store) ProjectHistory(repoID int64, cwdRel string, limit int) ([]Entry, error) {
	rows, err := s.db.Query(
		"SELECT id, repo_id, cwd_rel, intent, command, outcome, created_at FROM history WHERE repo_id = ? AND cwd_rel = ? ORDER BY created_at DESC LIMIT ?",
		repoID, cwdRel, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

func (s *Store) RecentGlobal(limit int) ([]Entry, error) {
	rows, err := s.db.Query(
		"SELECT id, repo_id, cwd_rel, intent, command, outcome, created_at FROM history WHERE outcome = ? ORDER BY created_at DESC LIMIT ?",
		OutcomeAccepted, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

func scanEntries(rows *sql.Rows) ([]Entry, error) {
	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.RepoID, &e.CwdRel, &e.Intent, &e.Command, &e.Outcome, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/history/ -v`
Expected: all tests PASS

**Step 6: Commit**

```bash
git add internal/history/
git commit -m "feat: add SQLite history store with migrations"
```

---

### Task 3: Context Parser Interface + Git Parser

**Files:**
- Create: `internal/context/parser.go`
- Create: `internal/context/git.go`
- Create: `internal/context/collect.go`
- Create: `internal/context/git_test.go`

**Step 1: Define the parser interface and context types**

Create `internal/context/parser.go`:

```go
package context

// Result holds the output of a single parser.
type Result struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data"`
}

// Snapshot is the full context collected from all parsers.
type Snapshot struct {
	RepoRoot string   `json:"repo_root"`
	CwdRel   string   `json:"cwd_rel"`
	Results  []Result `json:"results"`
}

// Parser detects project tooling and extracts relevant info.
// Returns nil result if this parser doesn't apply.
type Parser interface {
	Name() string
	Parse(repoRoot, cwd string) (*Result, error)
}
```

**Step 2: Write failing test for Git parser**

Create `internal/context/git_test.go`:

```go
package context

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitParser_InRepo(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0o644)
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "init")

	p := &GitParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.Data["repo_root"] != dir {
		t.Errorf("expected repo_root %q, got %q", dir, result.Data["repo_root"])
	}
	if _, ok := result.Data["branch"]; !ok {
		t.Error("expected branch in result")
	}
}

func TestGitParser_NotARepo(t *testing.T) {
	dir := t.TempDir()
	p := &GitParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for non-repo dir")
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./internal/context/ -run TestGitParser -v`
Expected: compilation error

**Step 4: Implement Git parser**

Create `internal/context/git.go`:

```go
package context

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitParser struct{}

func (g *GitParser) Name() string { return "git" }

func (g *GitParser) Parse(repoRoot, cwd string) (*Result, error) {
	if _, err := os.Stat(filepath.Join(repoRoot, ".git")); os.IsNotExist(err) {
		return nil, nil
	}

	data := map[string]any{
		"repo_root": repoRoot,
	}

	if branch, err := gitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		data["branch"] = branch
	}

	if changed, err := gitOutput(repoRoot, "diff", "--name-only", "HEAD"); err == nil && changed != "" {
		data["changed_files"] = strings.Split(changed, "\n")
	}

	return &Result{Name: g.Name(), Data: data}, nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
```

**Step 5: Implement the collector**

Create `internal/context/collect.go`:

```go
package context

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultParsers returns all v0 parsers.
func DefaultParsers() []Parser {
	return []Parser{
		&GitParser{},
		// More parsers added in subsequent tasks
	}
}

// Collect runs all parsers and builds a Snapshot.
func Collect(cwd string, parsers []Parser) (*Snapshot, error) {
	repoRoot := findRepoRoot(cwd)
	if repoRoot == "" {
		repoRoot = cwd
	}

	cwdRel, err := filepath.Rel(repoRoot, cwd)
	if err != nil {
		cwdRel = "."
	}

	snap := &Snapshot{
		RepoRoot: repoRoot,
		CwdRel:   cwdRel,
	}

	for _, p := range parsers {
		result, err := p.Parse(repoRoot, cwd)
		if err != nil {
			continue // skip parsers that fail
		}
		if result != nil {
			snap.Results = append(snap.Results, *result)
		}
	}

	return snap, nil
}

func findRepoRoot(cwd string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
```

**Step 6: Run tests to verify they pass**

Run: `go test ./internal/context/ -v`
Expected: all tests PASS

**Step 7: Commit**

```bash
git add internal/context/
git commit -m "feat: add context parser interface, git parser, and collector"
```

---

### Task 4: Node, Make, and Just Parsers

**Files:**
- Create: `internal/context/node.go`
- Create: `internal/context/makefile.go`
- Create: `internal/context/just.go`
- Create: `internal/context/node_test.go`
- Create: `internal/context/makefile_test.go`
- Create: `internal/context/just_test.go`
- Modify: `internal/context/collect.go` — register new parsers

**Step 1: Write failing tests**

Create `internal/context/node_test.go`:

```go
package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNodeParser_WithPackageJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{
		"scripts": {"test": "jest", "build": "tsc", "dev": "vite"}
	}`), 0o644)
	os.WriteFile(filepath.Join(dir, "bun.lockb"), []byte{}, 0o644)

	p := &NodeParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	scripts := result.Data["scripts"].([]string)
	if len(scripts) != 3 {
		t.Errorf("expected 3 scripts, got %d", len(scripts))
	}
	if result.Data["package_manager"] != "bun" {
		t.Errorf("expected bun, got %v", result.Data["package_manager"])
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
		t.Error("expected nil")
	}
}
```

Create `internal/context/makefile_test.go`:

```go
package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMakeParser_WithMakefile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Makefile"), []byte(
		".PHONY: test lint build\n\ntest:\n\tgo test ./...\n\nlint:\n\tgolangci-lint run\n\nbuild:\n\tgo build .\n",
	), 0o644)

	p := &MakeParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	targets := result.Data["targets"].([]string)
	if len(targets) != 3 {
		t.Errorf("expected 3 targets, got %d: %v", len(targets), targets)
	}
}
```

Create `internal/context/just_test.go`:

```go
package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJustParser_WithJustfile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Justfile"), []byte(
		"test:\n  go test ./...\n\nbuild:\n  go build .\n",
	), 0o644)

	p := &JustParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	recipes := result.Data["recipes"].([]string)
	if len(recipes) != 2 {
		t.Errorf("expected 2 recipes, got %d", len(recipes))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/context/ -v`
Expected: compilation errors

**Step 3: Implement Node parser**

Create `internal/context/node.go`:

```go
package context

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type NodeParser struct{}

func (n *NodeParser) Name() string { return "node" }

func (n *NodeParser) Parse(repoRoot, cwd string) (*Result, error) {
	pkgPath := findFileUpward("package.json", cwd, repoRoot)
	if pkgPath == "" {
		return nil, nil
	}

	data := map[string]any{}

	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Scripts    map[string]string `json:"scripts"`
		Workspaces []string         `json:"workspaces"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, err
	}

	if len(pkg.Scripts) > 0 {
		scripts := make([]string, 0, len(pkg.Scripts))
		for name := range pkg.Scripts {
			scripts = append(scripts, name)
		}
		data["scripts"] = scripts
	}

	if len(pkg.Workspaces) > 0 {
		data["workspaces"] = pkg.Workspaces
	}

	data["package_manager"] = detectPackageManager(repoRoot)

	return &Result{Name: n.Name(), Data: data}, nil
}

func detectPackageManager(dir string) string {
	lockfiles := map[string]string{
		"bun.lockb":        "bun",
		"bun.lock":         "bun",
		"pnpm-lock.yaml":   "pnpm",
		"yarn.lock":        "yarn",
		"package-lock.json": "npm",
	}
	for file, pm := range lockfiles {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			return pm
		}
	}
	return "npm"
}

// findFileUpward walks from start to root looking for filename.
func findFileUpward(filename, start, root string) string {
	dir := start
	for {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return path
		}
		if dir == root {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
```

**Step 4: Implement Make parser**

Create `internal/context/makefile.go`:

```go
package context

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type MakeParser struct{}

func (m *MakeParser) Name() string { return "make" }

func (m *MakeParser) Parse(repoRoot, cwd string) (*Result, error) {
	makePath := filepath.Join(repoRoot, "Makefile")
	if _, err := os.Stat(makePath); os.IsNotExist(err) {
		return nil, nil
	}

	targets, err := parseMakeTargets(makePath)
	if err != nil {
		return nil, err
	}

	if len(targets) == 0 {
		return nil, nil
	}

	return &Result{
		Name: m.Name(),
		Data: map[string]any{"targets": targets},
	}, nil
}

var makeTargetRe = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*)\s*:`)

func parseMakeTargets(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var targets []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "#") {
			continue
		}
		if m := makeTargetRe.FindStringSubmatch(line); m != nil {
			targets = append(targets, m[1])
		}
	}
	return targets, scanner.Err()
}
```

**Step 5: Implement Just parser**

Create `internal/context/just.go`:

```go
package context

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type JustParser struct{}

func (j *JustParser) Name() string { return "just" }

func (j *JustParser) Parse(repoRoot, cwd string) (*Result, error) {
	justPath := findJustfile(repoRoot)
	if justPath == "" {
		return nil, nil
	}

	recipes, err := parseJustRecipes(justPath)
	if err != nil {
		return nil, err
	}

	if len(recipes) == 0 {
		return nil, nil
	}

	return &Result{
		Name: j.Name(),
		Data: map[string]any{"recipes": recipes},
	}, nil
}

func findJustfile(dir string) string {
	for _, name := range []string{"Justfile", "justfile", ".justfile"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

var justRecipeRe = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*)\s*.*:`)

func parseJustRecipes(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var recipes []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "#") {
			continue
		}
		if m := justRecipeRe.FindStringSubmatch(line); m != nil {
			recipes = append(recipes, m[1])
		}
	}
	return recipes, scanner.Err()
}
```

**Step 6: Register parsers in collect.go**

Update `DefaultParsers()` in `internal/context/collect.go`:

```go
func DefaultParsers() []Parser {
	return []Parser{
		&GitParser{},
		&NodeParser{},
		&MakeParser{},
		&JustParser{},
	}
}
```

**Step 7: Run tests to verify they pass**

Run: `go test ./internal/context/ -v`
Expected: all tests PASS

**Step 8: Commit**

```bash
git add internal/context/
git commit -m "feat: add Node, Make, and Just context parsers"
```

---

### Task 5: Go, Docker, Python, and Scripts Parsers

**Files:**
- Create: `internal/context/golang.go`
- Create: `internal/context/docker.go`
- Create: `internal/context/python.go`
- Create: `internal/context/scripts.go`
- Create: `internal/context/golang_test.go`
- Create: `internal/context/docker_test.go`
- Create: `internal/context/python_test.go`
- Create: `internal/context/scripts_test.go`
- Modify: `internal/context/collect.go` — register new parsers

**Step 1: Write failing tests for all four parsers**

Create `internal/context/golang_test.go`:

```go
package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoParser_WithGoMod(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/example/app\n\ngo 1.25\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main"), 0o644)

	p := &GoParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Data["module"] != "github.com/example/app" {
		t.Errorf("unexpected module: %v", result.Data["module"])
	}
	if result.Data["has_tests"] != true {
		t.Error("expected has_tests=true")
	}
}
```

Create `internal/context/docker_test.go`:

```go
package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDockerParser_WithCompose(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(
		"services:\n  web:\n    image: nginx\n  db:\n    image: postgres:15\n",
	), 0o644)

	p := &DockerParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	services := result.Data["services"].([]string)
	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}
}
```

Create `internal/context/python_test.go`:

```go
package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonParser_WithPyproject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(
		"[tool.poetry]\nname = \"myapp\"\n",
	), 0o644)
	os.WriteFile(filepath.Join(dir, "poetry.lock"), []byte{}, 0o644)

	p := &PythonParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Data["package_manager"] != "poetry" {
		t.Errorf("expected poetry, got %v", result.Data["package_manager"])
	}
}
```

Create `internal/context/scripts_test.go`:

```go
package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScriptsParser_WithBinDir(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	os.MkdirAll(binDir, 0o755)
	os.WriteFile(filepath.Join(binDir, "deploy.sh"), []byte("#!/bin/bash"), 0o755)
	os.WriteFile(filepath.Join(binDir, "setup"), []byte("#!/bin/bash"), 0o755)

	p := &ScriptsParser{}
	result, err := p.Parse(dir, dir)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	scripts := result.Data["scripts"].([]string)
	if len(scripts) != 2 {
		t.Errorf("expected 2 scripts, got %d", len(scripts))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/context/ -v`
Expected: compilation errors

**Step 3: Implement Go parser**

Create `internal/context/golang.go`:

```go
package context

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type GoParser struct{}

func (g *GoParser) Name() string { return "go" }

func (g *GoParser) Parse(repoRoot, cwd string) (*Result, error) {
	modPath := filepath.Join(repoRoot, "go.mod")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		return nil, nil
	}

	data := map[string]any{}

	if mod, err := parseGoMod(modPath); err == nil {
		data["module"] = mod
	}

	data["has_tests"] = hasTestFiles(repoRoot)

	return &Result{Name: g.Name(), Data: data}, nil
}

func parseGoMod(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}
	return "", scanner.Err()
}

func hasTestFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	// Check one level of subdirectories
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			subEntries, err := os.ReadDir(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			for _, se := range subEntries {
				if !se.IsDir() && strings.HasSuffix(se.Name(), "_test.go") {
					return true
				}
			}
		}
	}
	return false
}
```

**Step 4: Implement Docker parser**

Create `internal/context/docker.go`:

```go
package context

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type DockerParser struct{}

func (d *DockerParser) Name() string { return "docker" }

func (d *DockerParser) Parse(repoRoot, cwd string) (*Result, error) {
	composePath := findComposeFile(repoRoot)
	if composePath == "" {
		return nil, nil
	}

	services, err := parseComposeServices(composePath)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, nil
	}

	return &Result{
		Name: d.Name(),
		Data: map[string]any{"services": services},
	}, nil
}

func findComposeFile(dir string) string {
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// Simple line-based parser — avoids pulling in a YAML library just for service names.
// Looks for top-level "services:" then indented service names.
var serviceNameRe = regexp.MustCompile(`^  ([a-zA-Z_][a-zA-Z0-9_-]*):\s*$`)

func parseComposeServices(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var services []string
	inServices := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "services:" {
			inServices = true
			continue
		}
		if inServices {
			// End of services block: a non-indented, non-empty line
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
				break
			}
			if m := serviceNameRe.FindStringSubmatch(line); m != nil {
				services = append(services, m[1])
			}
		}
	}
	return services, scanner.Err()
}
```

**Step 5: Implement Python parser**

Create `internal/context/python.go`:

```go
package context

import (
	"os"
	"path/filepath"
)

type PythonParser struct{}

func (p *PythonParser) Name() string { return "python" }

func (p *PythonParser) Parse(repoRoot, cwd string) (*Result, error) {
	indicators := []string{"pyproject.toml", "setup.py", "setup.cfg", "requirements.txt", "Pipfile"}
	found := false
	for _, name := range indicators {
		if _, err := os.Stat(filepath.Join(repoRoot, name)); err == nil {
			found = true
			break
		}
	}
	if !found {
		return nil, nil
	}

	data := map[string]any{
		"package_manager": detectPythonPM(repoRoot),
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "pyproject.toml")); err == nil {
		data["has_pyproject"] = true
	}

	// Detect virtual environment
	for _, venv := range []string{".venv", "venv", ".env"} {
		if info, err := os.Stat(filepath.Join(repoRoot, venv)); err == nil && info.IsDir() {
			data["venv"] = venv
			break
		}
	}

	return &Result{Name: p.Name(), Data: data}, nil
}

func detectPythonPM(dir string) string {
	lockfiles := map[string]string{
		"poetry.lock": "poetry",
		"uv.lock":     "uv",
		"Pipfile.lock": "pipenv",
		"pdm.lock":    "pdm",
	}
	for file, pm := range lockfiles {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			return pm
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		return "pip"
	}
	return "pip"
}
```

**Step 6: Implement Scripts parser**

Create `internal/context/scripts.go`:

```go
package context

import (
	"os"
	"path/filepath"
)

type ScriptsParser struct{}

func (s *ScriptsParser) Name() string { return "scripts" }

func (s *ScriptsParser) Parse(repoRoot, cwd string) (*Result, error) {
	var scripts []string

	for _, dirName := range []string{"bin", "scripts"} {
		dir := filepath.Join(repoRoot, dirName)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				scripts = append(scripts, filepath.Join(dirName, e.Name()))
			}
		}
	}

	if len(scripts) == 0 {
		return nil, nil
	}

	return &Result{
		Name: s.Name(),
		Data: map[string]any{"scripts": scripts},
	}, nil
}
```

**Step 7: Register all parsers in collect.go**

Update `DefaultParsers()`:

```go
func DefaultParsers() []Parser {
	return []Parser{
		&GitParser{},
		&NodeParser{},
		&MakeParser{},
		&JustParser{},
		&GoParser{},
		&DockerParser{},
		&PythonParser{},
		&ScriptsParser{},
	}
}
```

**Step 8: Run tests to verify they pass**

Run: `go test ./internal/context/ -v`
Expected: all tests PASS

**Step 9: Commit**

```bash
git add internal/context/
git commit -m "feat: add Go, Docker, Python, and Scripts context parsers"
```

---

### Task 6: Config Loading

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Install TOML dependency**

Run: `go get github.com/BurntSushi/toml@latest`

**Step 2: Write failing test**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	os.WriteFile(cfgPath, []byte(`
[llm]
api_key = "sk-ant-test"

[llm.models]
fast = "claude-haiku-4-5-20251001"
strong = "claude-sonnet-4-5-20250929"
escalation_threshold = 0.8
`), 0o644)

	cfg, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.APIKey != "sk-ant-test" {
		t.Errorf("unexpected api_key: %q", cfg.LLM.APIKey)
	}
	if cfg.LLM.Models.EscalationThreshold != 0.8 {
		t.Errorf("unexpected threshold: %v", cfg.LLM.Models.EscalationThreshold)
	}
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.Models.Fast != "claude-haiku-4-5-20251001" {
		t.Errorf("unexpected default fast model: %q", cfg.LLM.Models.Fast)
	}
	if cfg.LLM.Models.EscalationThreshold != 0.7 {
		t.Errorf("unexpected default threshold: %v", cfg.LLM.Models.EscalationThreshold)
	}
}

func TestLoad_EnvOverridesKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-from-env")
	cfg, err := LoadFrom("/nonexistent/path")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.APIKey != "sk-from-env" {
		t.Errorf("expected env override, got %q", cfg.LLM.APIKey)
	}
}
```

**Step 3: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`
Expected: compilation errors

**Step 4: Implement config**

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

type Config struct {
	LLM LLMConfig `toml:"llm"`
}

type LLMConfig struct {
	APIKey string      `toml:"api_key"`
	Models ModelsConfig `toml:"models"`
}

type ModelsConfig struct {
	Fast                 string  `toml:"fast"`
	Strong               string  `toml:"strong"`
	EscalationThreshold  float64 `toml:"escalation_threshold"`
}

func defaults() Config {
	return Config{
		LLM: LLMConfig{
			Models: ModelsConfig{
				Fast:                "claude-haiku-4-5-20251001",
				Strong:              "claude-sonnet-4-5-20250929",
				EscalationThreshold: 0.7,
			},
		},
	}
}

func Load() (*Config, error) {
	return LoadFrom(defaultPath())
}

func LoadFrom(path string) (*Config, error) {
	cfg := defaults()

	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return nil, err
		}
	}

	// Env var overrides config file
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}

	return &cfg, nil
}

func defaultPath() string {
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "pls", "config.toml")
	}
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		home, _ := os.UserHomeDir()
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "pls", "config.toml")
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: all tests PASS

**Step 6: Commit**

```bash
git add internal/config/ go.mod go.sum
git commit -m "feat: add TOML config loading with defaults and env override"
```

---

### Task 7: LLM Client + Prompt Building

**Files:**
- Create: `internal/llm/types.go`
- Create: `internal/llm/prompt.go`
- Create: `internal/llm/client.go`
- Create: `internal/llm/prompt_test.go`

**Step 1: Install Anthropic SDK**

Run: `go get github.com/anthropics/anthropic-sdk-go@latest`

**Step 2: Define types**

Create `internal/llm/types.go`:

```go
package llm

type Candidate struct {
	Cmd        string  `json:"cmd"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
	Risk       string  `json:"risk"` // "safe", "moderate", "dangerous"
}

type Response struct {
	Candidates []Candidate `json:"candidates"`
}
```

**Step 3: Write failing test for prompt building**

Create `internal/llm/prompt_test.go`:

```go
package llm

import (
	"strings"
	"testing"

	"github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
)

func TestBuildPrompt_IncludesIntent(t *testing.T) {
	prompt := BuildPrompt("run the tests", &context.Snapshot{
		RepoRoot: "/project",
		CwdRel:   ".",
	}, nil, nil)

	if !strings.Contains(prompt, "run the tests") {
		t.Error("prompt should contain intent")
	}
}

func TestBuildPrompt_IncludesContext(t *testing.T) {
	snap := &context.Snapshot{
		RepoRoot: "/project",
		CwdRel:   "apps/web",
		Results: []context.Result{
			{Name: "node", Data: map[string]any{"scripts": []string{"test", "build"}, "package_manager": "bun"}},
		},
	}
	prompt := BuildPrompt("run tests", snap, nil, nil)

	if !strings.Contains(prompt, "bun") {
		t.Error("prompt should include package manager")
	}
	if !strings.Contains(prompt, "apps/web") {
		t.Error("prompt should include cwd_rel")
	}
}

func TestBuildPrompt_IncludesHistory(t *testing.T) {
	projectHistory := []history.Entry{
		{Intent: "run tests", Command: "bun test", Outcome: "accepted"},
		{Intent: "lint code", Command: "eslint .", Outcome: "rejected"},
	}
	prompt := BuildPrompt("run tests", &context.Snapshot{RepoRoot: "/p", CwdRel: "."}, projectHistory, nil)

	if !strings.Contains(prompt, "bun test") {
		t.Error("prompt should include project history commands")
	}
	if !strings.Contains(prompt, "rejected") {
		t.Error("prompt should include rejection info")
	}
}

func TestSystemPrompt_IsReasonable(t *testing.T) {
	sys := SystemPrompt()
	if !strings.Contains(sys, "candidates") {
		t.Error("system prompt should mention candidates")
	}
	if !strings.Contains(sys, "risk") {
		t.Error("system prompt should mention risk")
	}
}
```

**Step 4: Run tests to verify they fail**

Run: `go test ./internal/llm/ -v`
Expected: compilation errors

**Step 5: Implement prompt building**

Create `internal/llm/prompt.go`:

```go
package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
)

func SystemPrompt() string {
	return `You are a shell command generator. Given a project context and user intent, generate candidate shell commands.

Rules:
- Return valid JSON only, no markdown fences, no prose
- Return 1-5 candidate commands, ranked by confidence
- Commands can include pipes, chaining, jq, docker exec, psql, etc.
- Prefer commands grounded in the project context (detected tools, scripts, services)
- Use the command history to learn user preferences (e.g., if they prefer rg over grep)
- Avoid repeating commands the user previously rejected for the same intent
- Classify each command's risk level

Response schema:
{
  "candidates": [
    {
      "cmd": "the shell command",
      "reason": "short explanation of why this command",
      "confidence": 0.0 to 1.0,
      "risk": "safe|moderate|dangerous"
    }
  ]
}

Risk levels:
- "safe": read-only, informational (ls, cat, grep, select queries, status)
- "moderate": writes data but reversible (git commit, insert, create file, build)
- "dangerous": destructive or hard to reverse (rm, drop, truncate, force push, chmod -R)`
}

func BuildPrompt(intent string, snap *context.Snapshot, projectHistory []history.Entry, globalHistory []history.Entry) string {
	var b strings.Builder

	b.WriteString("## Project Context\n\n")
	b.WriteString(fmt.Sprintf("Working directory: %s (relative: %s)\n\n", snap.RepoRoot, snap.CwdRel))

	if len(snap.Results) > 0 {
		contextJSON, _ := json.MarshalIndent(snap.Results, "", "  ")
		b.WriteString("Detected tooling:\n")
		b.WriteString(string(contextJSON))
		b.WriteString("\n\n")
	}

	if len(globalHistory) > 0 {
		b.WriteString("## Recent Commands (global, across all projects)\n\n")
		for _, e := range globalHistory {
			b.WriteString(fmt.Sprintf("- \"%s\" → `%s`\n", e.Intent, e.Command))
		}
		b.WriteString("\n")
	}

	if len(projectHistory) > 0 {
		b.WriteString("## Command History (this project + directory)\n\n")
		for _, e := range projectHistory {
			b.WriteString(fmt.Sprintf("- \"%s\" → `%s` [%s]\n", e.Intent, e.Command, e.Outcome))
		}
		b.WriteString("\n")
	}

	b.WriteString("## User Intent\n\n")
	b.WriteString(intent)

	return b.String()
}
```

**Step 6: Implement the LLM client**

Create `internal/llm/client.go`:

```go
package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/dpup/pls/internal/config"
	"github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
)

type Client struct {
	api    anthropic.Client
	config *config.Config
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		api:    anthropic.NewClient(option.WithAPIKey(cfg.LLM.APIKey)),
		config: cfg,
	}
}

// Generate produces command candidates for the given intent.
// Uses the fast model first, escalates to strong if confidence is low.
func (c *Client) Generate(
	ctx context.Snapshot,
	intent string,
	projectHistory []history.Entry,
	globalHistory []history.Entry,
) (*Response, error) {
	prompt := BuildPrompt(intent, &ctx, projectHistory, globalHistory)

	// Try fast model first
	resp, err := c.call(c.config.LLM.Models.Fast, prompt)
	if err != nil {
		return nil, err
	}

	// Escalate if low confidence
	if len(resp.Candidates) > 0 && resp.Candidates[0].Confidence < c.config.LLM.Models.EscalationThreshold {
		strongResp, err := c.call(c.config.LLM.Models.Strong, prompt)
		if err == nil && len(strongResp.Candidates) > 0 {
			return strongResp, nil
		}
	}

	return resp, nil
}

// Escalate calls the strong model directly (used when user exhausts fast model candidates).
func (c *Client) Escalate(
	ctx context.Snapshot,
	intent string,
	projectHistory []history.Entry,
	globalHistory []history.Entry,
) (*Response, error) {
	prompt := BuildPrompt(intent, &ctx, projectHistory, globalHistory)
	return c.call(c.config.LLM.Models.Strong, prompt)
}

func (c *Client) call(model, prompt string) (*Response, error) {
	msg, err := c.api.Messages.New(anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: SystemPrompt()},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewTextBlock(prompt),
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic API: %w", err)
	}

	// Extract text from response
	var text string
	for _, block := range msg.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	// Strip markdown fences if present
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var resp Response
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("parsing LLM response: %w", err)
	}

	return &resp, nil
}
```

**Step 7: Run tests to verify they pass**

Run: `go test ./internal/llm/ -v`
Expected: prompt tests PASS (client tests require API key, tested via integration)

**Step 8: Commit**

```bash
git add internal/llm/ go.mod go.sum
git commit -m "feat: add LLM client with two-tier model escalation and prompt builder"
```

---

### Task 8: Command Execution

**Files:**
- Create: `internal/exec/runner.go`
- Create: `internal/exec/runner_test.go`

**Step 1: Write failing test**

Create `internal/exec/runner_test.go`:

```go
package exec

import (
	"bytes"
	"testing"
)

func TestRun_CapturesOutput(t *testing.T) {
	var stdout bytes.Buffer
	err := Run("echo hello", &stdout)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stdout.String() != "hello\n" {
		t.Errorf("unexpected output: %q", stdout.String())
	}
}

func TestRun_ReturnsError(t *testing.T) {
	var stdout bytes.Buffer
	err := Run("false", &stdout)
	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestRun_HandlesPipes(t *testing.T) {
	var stdout bytes.Buffer
	err := Run("echo hello world | tr ' ' '\\n' | wc -l", &stdout)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/exec/ -v`
Expected: compilation errors

**Step 3: Implement runner**

Create `internal/exec/runner.go`:

```go
package exec

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Run executes a command string via the user's shell, streaming output to w.
// Uses $SHELL if set, falls back to /bin/sh.
func Run(command string, w io.Writer) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-c", command)
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/exec/ -v`
Expected: all tests PASS

**Step 5: Commit**

```bash
git add internal/exec/
git commit -m "feat: add command runner with shell execution"
```

---

### Task 9: TUI with bubbletea + lipgloss

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/model.go`

**Step 1: Install dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/atotto/clipboard@latest
```

**Step 2: Implement styles**

Create `internal/tui/styles.go`:

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	commandStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		PaddingLeft(2)

	reasonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		PaddingLeft(2)

	riskSafe = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	riskModerate = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	riskDangerous = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		PaddingLeft(2).
		PaddingTop(1)

	keyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Bold(true)
)

func riskStyle(risk string) lipgloss.Style {
	switch risk {
	case "dangerous":
		return riskDangerous
	case "moderate":
		return riskModerate
	default:
		return riskSafe
	}
}

func riskLabel(risk string) string {
	switch risk {
	case "dangerous":
		return "■ dangerous"
	case "moderate":
		return "■ moderate"
	default:
		return "■ safe"
	}
}
```

**Step 3: Implement the bubbletea model**

Create `internal/tui/model.go`:

```go
package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	plsexec "github.com/dpup/pls/internal/exec"
	"github.com/dpup/pls/internal/llm"
)

type Action int

const (
	ActionNone Action = iota
	ActionRun
	ActionCopy
	ActionQuit
)

type Result struct {
	Action    Action
	Candidate llm.Candidate
}

type Model struct {
	candidates []llm.Candidate
	index      int
	result     Result
	done       bool
	err        error
}

func New(candidates []llm.Candidate) Model {
	return Model{
		candidates: candidates,
		index:      0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y":
			m.result = Result{Action: ActionRun, Candidate: m.candidates[m.index]}
			m.done = true
			return m, tea.Quit
		case "c":
			m.result = Result{Action: ActionCopy, Candidate: m.candidates[m.index]}
			m.done = true
			return m, tea.Quit
		case "n":
			if m.index < len(m.candidates)-1 {
				m.index++
			}
		case "p":
			if m.index > 0 {
				m.index--
			}
		case "q", "ctrl+c":
			m.result = Result{Action: ActionQuit}
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.done || len(m.candidates) == 0 {
		return ""
	}

	c := m.candidates[m.index]
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(commandStyle.Render(c.Cmd))
	b.WriteString("\n\n")
	b.WriteString(reasonStyle.Render("Reason: " + c.Reason))
	b.WriteString("\n")
	b.WriteString(reasonStyle.Render("Risk:   ") + riskStyle(c.Risk).Render(riskLabel(c.Risk)))

	if len(m.candidates) > 1 {
		counter := reasonStyle.Render(fmt.Sprintf("        [%d/%d]", m.index+1, len(m.candidates)))
		b.WriteString(counter)
	}

	b.WriteString("\n")

	help := fmt.Sprintf(
		"%s run  %s copy  %s next  %s prev  %s quit",
		keyStyle.Render("[y]"),
		keyStyle.Render("[c]"),
		keyStyle.Render("[n]"),
		keyStyle.Render("[p]"),
		keyStyle.Render("[q]"),
	)
	b.WriteString(helpStyle.Render(help))
	b.WriteString("\n")

	return b.String()
}

func (m Model) Result() Result {
	return m.result
}

// RunTUI starts the interactive TUI and returns the user's choice.
func RunTUI(candidates []llm.Candidate) (*Result, error) {
	if len(candidates) == 0 {
		return &Result{Action: ActionQuit}, nil
	}

	model := New(candidates)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(Model).Result()
	return &result, nil
}

// Execute handles the result of the TUI interaction.
func Execute(result *Result) error {
	switch result.Action {
	case ActionRun:
		fmt.Fprintf(os.Stderr, "\n")
		return plsexec.Run(result.Candidate.Cmd, os.Stdout)
	case ActionCopy:
		if err := clipboard.WriteAll(result.Candidate.Cmd); err != nil {
			// Fallback: print the command for manual copying
			fmt.Fprintf(os.Stderr, "Copied to clipboard failed, command: %s\n", result.Candidate.Cmd)
			return nil
		}
		fmt.Fprintf(os.Stderr, "Copied to clipboard.\n")
		return nil
	default:
		return nil
	}
}
```

**Step 4: Verify it compiles**

Run: `go build ./internal/tui/`
Expected: compiles without errors

**Step 5: Commit**

```bash
git add internal/tui/ go.mod go.sum
git commit -m "feat: add interactive TUI with bubbletea and lipgloss styling"
```

---

### Task 10: Wire Everything Together

**Files:**
- Modify: `cmd/root.go` — integrate all layers
- Modify: `main.go` — no changes expected

**Step 1: Update cmd/root.go to wire the full pipeline**

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dpup/pls/internal/config"
	plsctx "github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
	"github.com/dpup/pls/internal/llm"
	"github.com/dpup/pls/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "pls [intent]",
	Short: "Project-aware natural language shell command router",
	Long:  "Translates natural language into the right shell command for your current project.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  run,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	intent := strings.Join(args, " ")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("no API key configured. Set ANTHROPIC_API_KEY or add api_key to config")
	}

	// Collect context
	cwd, _ := os.Getwd()
	snap, err := plsctx.Collect(cwd, plsctx.DefaultParsers())
	if err != nil {
		return fmt.Errorf("collecting context: %w", err)
	}

	// Open history store
	store, err := history.Open(historyDBPath())
	if err != nil {
		return fmt.Errorf("opening history: %w", err)
	}
	defer store.Close()

	// Query history
	repoID, _ := store.EnsureRepo(snap.RepoRoot)
	projectHistory, _ := store.ProjectHistory(repoID, snap.CwdRel, 20)
	globalHistory, _ := store.RecentGlobal(10)

	// Generate candidates
	client := llm.NewClient(cfg)
	resp, err := client.Generate(*snap, intent, projectHistory, globalHistory)
	if err != nil {
		return fmt.Errorf("generating commands: %w", err)
	}
	if len(resp.Candidates) == 0 {
		fmt.Println("No command candidates generated.")
		return nil
	}

	// Run TUI
	result, err := tui.RunTUI(resp.Candidates)
	if err != nil {
		return fmt.Errorf("TUI: %w", err)
	}

	// Record outcome
	switch result.Action {
	case tui.ActionRun:
		store.Record(repoID, snap.CwdRel, intent, result.Candidate.Cmd, history.OutcomeAccepted)
	case tui.ActionCopy:
		store.Record(repoID, snap.CwdRel, intent, result.Candidate.Cmd, history.OutcomeCopied)
	}

	// Execute the action
	return tui.Execute(result)
}

func historyDBPath() string {
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "pls", "history.db")
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "pls", "history.db")
}
```

**Step 2: Run go mod tidy and verify it builds**

Run:
```bash
go mod tidy
go build -o pls .
```
Expected: compiles without errors

**Step 3: Smoke test**

Run: `ANTHROPIC_API_KEY=test ./pls "hello"` (will fail on API call but verifies wiring)
Expected: error about API call, not about wiring/nil pointers

**Step 4: Commit**

```bash
git add cmd/ go.mod go.sum
git commit -m "feat: wire context, LLM, history, and TUI into root command"
```

---

### Task 11: End-to-End Manual Test

**Step 1: Build and install**

Run:
```bash
go build -o pls .
```

**Step 2: Run with a real API key**

Run: `./pls "list files"`
Expected: TUI appears with a command candidate, risk coloring, and key bindings

**Step 3: Test each key binding**

- Press `n`/`p` to cycle candidates
- Press `c` to copy
- Press `q` to quit
- Press `y` to run

**Step 4: Verify history is recorded**

Run: `./pls "list files"` again
Expected: LLM should receive history from previous run

**Step 5: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix: address issues found during manual testing"
```
