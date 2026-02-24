# pls

Describe what you want. Get the exact command for your project.

```
$ pls run just the history tests

  ╭─────────────────────────────────────────────────────────────────────╮
  │ go test ./internal/history -v -run TestRecordAndQueryProjectHistory │
  ╰─────────────────────────────────────────────────────────────────────╯

  Reason: Runs the specific test that checks history recording
  Risk:   ■ safe        [1/2]

  [y] run  [c] copy  [n] next  [p] prev  [q] quit
```

`pls` reads your project — build tools, package managers, Makefiles, test layout, and your command history — then uses an LLM to suggest precise shell commands grounded in what's actually there.

Deterministic parsers collect your project's real affordances quickly. The model reasons over that structured context to map your intent to a concrete command. When it needs specifics — an exact test name or config value — it can read your files directly.

## Why

I work on a lot of different projects across different ecosystems — jumping from yarn to npm to bun to Make to poetry — and I can never remember the right incantation. I realized that when I was inside a Claude Code session I never had that problem; the model just knew. I wanted something faster and lighter weight for those moments when I just need the command, without spinning up a full coding session.

## Install

Download a prebuilt binary from [Releases](https://github.com/dpup/pls/releases), or install with Go:

```bash
go install github.com/dpup/pls@latest
```

Or build from source:

```bash
git clone https://github.com/dpup/pls.git
cd pls
make build
```

## Setup

Set your Anthropic API key:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

Or add it to a config file:

```toml
# macOS: ~/Library/Application Support/pls/config.toml
# Linux: ~/.config/pls/config.toml

[llm]
api_key = "sk-ant-..."

[llm.models]
fast   = "claude-haiku-4-5-20251001"   # default
strong = "claude-sonnet-4-5-20250929"  # used for low-confidence escalation
escalation_threshold = 0.7
```

## Usage

```bash
# Basic usage — interactive TUI
pls "deploy to staging"

# JSON output for scripting
pls --json "run the linter"

# See what context the LLM receives
pls --verbose "run tests"

# See the full prompt without calling the API (no key needed)
pls --explain "run tests for history"
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--json` | | Output candidates as JSON |
| `--verbose` | `-v` | Print detected context and tool-use log |
| `--explain` | | Print the full LLM prompt, then exit (no API call) |
| `--version` | | Print version |

## How it works

### Context detection

Before calling the LLM, `pls` scans your project to build a structured snapshot of your tooling:

| What it detects | Examples |
|----------------|---------|
| **Git** | Branch, changed files, repo root |
| **Go** | Module name, packages, which packages have tests |
| **Node** | Package manager (npm/yarn/pnpm/bun), scripts |
| **Make** | Makefile targets |
| **Just** | Justfile recipes |
| **Docker** | Compose services |
| **Python** | Package manager (pip/poetry/uv), virtualenv |
| **Scripts** | Executables in `bin/` and `scripts/` |

This context is sent to the LLM so it can suggest commands that use your actual tools — not generic guesses.

### File exploration

For specific intents (like targeting a single test by name), static context isn't enough. The LLM can explore your project via sandboxed `list_files` and `read_file` tool calls to find exact file paths, function names, and config details before suggesting a command.

This is capped at 2 tool rounds to keep response times fast.

### Risk classification

Every candidate is classified as **safe**, **moderate**, or **dangerous** so you can see at a glance whether a command is read-only (`git status`), a reversible write (`git commit`), or destructive (`rm -rf`).

### Model escalation

`pls` uses Claude Haiku for speed. If the top candidate's confidence is below the escalation threshold (default 0.7), it automatically retries with Claude Sonnet for a more thorough answer.

### Command history

Every command you accept or copy is recorded in a local SQLite database. This history is fed back into future prompts so the LLM learns your preferences — if you always use `make test` instead of `go test ./...`, it picks that up. Rejected commands are marked so they won't be suggested again.

## Why not just use Claude Code?

`claude -p` can suggest commands too, but it infers project structure from scratch every time. `pls` front-loads that work with fast, deterministic parsers and feeds the LLM a compact, structured snapshot — so it gives more project-aware suggestions.

Informal comparison on a Go project with a Makefile (~130k LOC). All three use Claude Haiku:

**"run the e2e tests"**
```
pls (3.2s)       →  make test-e2e                                        ✅ correct
claude -p (3.5s) →  go test -tags=e2e -v ./internal/e2e/                 ⚠️ works, missed Makefile target
```

**"run the logging e2e tests"**
```
pls (3.2s)       →  make test-e2e ARGS='-run Logs'                       ✅ correct
claude -p (5.0s) →  go test -tags=e2e -v ./internal/e2e -run TestLog     ❌ fabricated test name, runs nothing
```

**"run the tests"**
```
pls (3.3s)       →  make test  (+ 2 alternatives)                        ✅ correct
claude -p (3.5s) →  go test ./...                                        ⚠️ works, missed Makefile target
```

**"run the config tests"**
```
pls (3.6s)       →  go test ./internal/config -v                         ✅ correct (Makefile target as option 2)
claude -p (3.9s) →  go test ./internal/config/...                        ⚠️ works, missed Makefile target
```

`pls` uses Makefile targets and real test names because its parsers detect them before the LLM runs. `claude -p` infers structure from raw file access and often falls back to generic commands — or fabricates names that don't exist. And because `pls` records which commands you accept, it learns your preferences over time — if you pick the Makefile target once, it'll rank it first next time.

## Development

```bash
make test       # run tests
make lint       # run golangci-lint + go vet
make fix        # auto-format, tidy modules
make security   # run govulncheck
make build      # compile with version injection
```

### Adding a new context parser

Implement the `Parser` interface in `internal/context/`:

```go
type Parser interface {
    Name() string
    Parse(repoRoot, cwd string) (*Result, error)
}
```

Then register it in `DefaultParsers()` in `collect.go`.

## Platform support

macOS and Linux. Windows is not currently supported.

## License

[MIT](LICENSE)
