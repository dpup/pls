# pls

Project-aware natural language shell command router. Translates what you *want* to do into the right command for your current project.

```
$ pls "run just the history tests"

  ╭─────────────────────────────────────────────────────────────────────╮
  │ go test ./internal/history -v -run TestRecordAndQueryProjectHistory │
  ╰─────────────────────────────────────────────────────────────────────╯

  Reason: Runs the specific test that checks history recording
  Risk:   ■ safe        [1/2]

  [y] run  [c] copy  [n] next  [p] prev  [q] quit
```

`pls` understands your project. It detects your build tools, package managers, Makefiles, test structure, and command history — then uses Claude to suggest precise commands grounded in that context. When needed, the LLM can explore your project via tool calls to find exact file paths, test names, and config details.

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

`pls` scans your project and detects:

| Parser | What it finds |
|--------|--------------|
| **git** | Branch, changed files, repo root |
| **go** | Module name, all package paths, which packages have tests |
| **node** | Package manager (npm/yarn/pnpm/bun), scripts |
| **make** | Makefile targets |
| **just** | Justfile recipes |
| **docker** | Compose services |
| **python** | Package manager (pip/poetry/uv), virtualenv |
| **scripts** | Executables in `bin/` and `scripts/` |

### Tool-use loop

For specific intents (like targeting a particular test), the LLM can explore your project before answering:

1. Receives project context + your intent
2. Optionally calls `list_files` or `read_file` to inspect specific paths
3. Produces candidates grounded in what it actually found

This is capped at 2 tool rounds to keep response times fast.

### Model escalation

By default, `pls` uses Claude Haiku for speed. If the top candidate's confidence is below the escalation threshold (default 0.7), it automatically retries with Claude Sonnet.

### Command history

Every command you accept or copy is recorded in a local SQLite database. This history is included in future prompts so the LLM can learn your preferences and avoid repeating rejected commands.

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
