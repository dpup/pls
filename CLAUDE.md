# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is this?

`pls` is a Go CLI tool that translates natural language intent into shell commands using the Anthropic Claude API. It detects project context (build tools, package managers, test layout, command history) and uses an LLM to suggest precise commands grounded in what's actually present.

## Commands

```bash
make build      # compile binary with version injection via -ldflags
make test       # go test ./...
make lint       # golangci-lint run ./... (includes go vet)
make vet        # go vet ./... only
make fix        # gofmt -w . && go mod tidy
make security   # govulncheck ./...
```

Run a single test:
```bash
go test ./internal/history -run TestRecordAndQueryProjectHistory -v
```

## Architecture

The request pipeline flows through these stages:

1. **CLI** (`cli/root.go`) — Cobra command parses flags and intent, orchestrates the pipeline
2. **Config** (`internal/config/`) — Loads TOML config from OS-appropriate path, with env var fallback (`ANTHROPIC_API_KEY`)
3. **Context Collection** (`internal/context/`) — Runs deterministic parsers that detect project tooling (Git, Go, Node, Make, Just, Docker, Python, Scripts)
4. **History** (`internal/history/`) — SQLite store tracking intent→command→outcome per repo+directory, fed back into prompts so the LLM learns user preferences
5. **LLM** (`internal/llm/`) — Builds a prompt from context+history+intent, calls Claude Haiku with tool-use (sandboxed `list_files`/`read_file`), escalates to Sonnet if top candidate confidence < threshold (default 0.7)
6. **TUI** (`internal/tui/`) — Bubbletea interactive display showing candidates with risk labels; supports run/copy/navigate/quit

Entry point: `main.go` → `cli.Execute(version)`.

### Context Parsers

Each parser implements the `Parser` interface (`internal/context/parser.go`):
```go
type Parser interface {
    Name() string
    Parse(repoRoot, cwd string) (*Result, error)
}
```

Parsers are registered in `DefaultParsers()` in `collect.go`. To add a new parser: implement the interface, add to `DefaultParsers()`.

### LLM Tool Loop

The fast model gets up to 2 tool rounds (`maxToolTurns`) to explore the repo via `list_files` and `read_file` before producing a final JSON response. Tools are sandboxed to the repo root. On the final turn, tools are omitted to force a text response.

### Key Types

- `context.Snapshot` — collected project context (repo root, relative cwd, parser results)
- `llm.Candidate` — a suggested command with confidence score, reason, and risk level (safe/moderate/dangerous)
- `llm.Response` — list of candidates plus tool-use rounds
- `history.Entry` — recorded intent→command→outcome tuple

## Conventions

- Package imports use `plsctx` and `plsexec` aliases to avoid collision with stdlib `context` and `exec`
- Config paths follow XDG on Linux, `~/Library/Application Support/` on macOS
- History DB path follows the same OS convention under a `pls/` subdirectory
- CGO is disabled for release builds (uses `modernc.org/sqlite` pure-Go SQLite)
