# pls — Project-Aware Natural Language Shell Command Router

**Go + Cobra | Anthropic Claude | bubbletea + lipgloss**

## What it is

A CLI tool that translates natural language into the right shell command for
your current project. It detects project tooling, learns from your command
history, and lets you run commands directly.

```
$ pls "run the frontend tests"

  bun test --filter frontend

  Reason: bun.lockb present, test script in package.json
  Risk: safe

  [y] run  [c] copy  [n] next  [p] prev  [q] quit
```

## User Stories

1. **As a developer working across multiple repos**, I want to describe what I
   want to do in plain English and get the right shell command for this project,
   so I don't have to remember whether it's `bun test`, `make test`, or
   `go test ./...`.

2. **As a developer running complex commands**, I want to say "query the users
   table" and get `docker exec -it db psql -U app -c "SELECT * FROM users"`
   with the right container name and credentials for this project.

3. **As a developer who repeats commands**, I want pls to remember what worked
   last time and use that history to give better suggestions, without me having
   to configure anything.

4. **As a developer reviewing a suggested command**, I want to see a color-coded
   risk assessment so I can quickly tell if a command is safe, modifies state,
   or is destructive.

5. **As a developer**, I want to press `y` to run, `c` to copy, `n`/`p` to
   browse alternatives, or `q` to bail.

## Architecture

Three layers, each doing one job:

```
Intent ──> Context Collection ──> LLM Layer ──> TUI
                                     ^
                                     |
                                  History
                                 (SQLite)
```

### 1. Context Collection

Structured parsers detect project tooling and produce a context snapshot. Each
parser implements a common interface and returns nil if it doesn't apply. Parsers
walk from cwd up to repo root.

Runs every invocation (file existence checks and light parsing — fast enough to
not need caching).

#### v0 Parsers

| Parser  | Detects                                          | Extracts                                                 |
|---------|--------------------------------------------------|----------------------------------------------------------|
| Node    | `package.json`, lockfiles                        | scripts, package manager (npm/yarn/pnpm/bun), workspaces |
| Make    | `Makefile`                                       | target names                                             |
| Just    | `Justfile`                                       | recipe names                                             |
| Go      | `go.mod`                                         | module name, has `_test.go` files                        |
| Docker  | `docker-compose.yml`                             | service names, images                                    |
| Git     | `.git`                                           | repo root, branch, changed files                         |
| Python  | `pyproject.toml`, `setup.py`, `requirements.txt` | package manager (pip/poetry/uv), scripts, virtual env    |
| Scripts | `bin/`, `scripts/` directories                   | script filenames                                         |

Adding a new parser: implement the interface, register it. No config needed.

### 2. LLM Layer

#### Prompt Structure

1. System prompt: generate shell commands from project context and user intent
2. Project context (from parsers)
3. History: recent global commands (~10) + project-specific commands (~20)
4. User's intent

#### Response Format

```json
{
  "candidates": [
    {
      "cmd": "docker exec -it app-db psql -U postgres -c 'SELECT * FROM users'",
      "reason": "docker-compose has a db service running postgres",
      "confidence": 0.9,
      "risk": "read"
    }
  ]
}
```

#### Risk Classification

The LLM classifies each command:

- **safe** (green) — read-only, informational (ls, cat, grep, select queries)
- **moderate** (yellow) — writes data but reversible (git commit, insert, create file)
- **dangerous** (red) — destructive or hard to reverse (rm, drop, truncate, force push)

#### Two-Tier Model Escalation

1. Haiku generates candidates
2. If top candidate confidence < 0.7, re-run with Sonnet
3. If user presses `n` and Haiku's candidates are exhausted, escalate to Sonnet

### 3. History Store (SQLite)

No scoring engine. No intent normalization. No candidate ranking. Just a log
the LLM reads as context.

**Location:** `~/.local/share/pls/history.db` (Linux),
`~/Library/Application Support/pls/history.db` (macOS)

#### Schema

```sql
CREATE TABLE repos (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    root_path  TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE history (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id    INTEGER REFERENCES repos(id),
    cwd_rel    TEXT NOT NULL,
    intent     TEXT NOT NULL,
    command    TEXT NOT NULL,
    outcome    TEXT NOT NULL CHECK (outcome IN ('accepted', 'rejected', 'copied')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Context Queries

When building the LLM prompt, two queries:

1. **Recent global** — last ~10 accepted commands across all repos (captures
   general preferences like `rg` over `grep`)
2. **Project history** — last ~20 commands for this `(repo_id, cwd_rel)`
   including rejections (so the LLM avoids repeating rejected suggestions)

### 4. TUI (bubbletea + lipgloss)

Interactive single-command display with risk coloring.

```
$ pls "run the frontend tests"

  bun test --filter frontend

  Reason: bun.lockb present, test script in package.json
  Risk: ■ safe

  [y] run  [c] copy  [n] next  [p] prev  [q] quit
```

| Key | Action                                                        |
|-----|---------------------------------------------------------------|
| `y` | Execute command, stream output, record as `accepted`          |
| `c` | Copy to clipboard, record as `copied`                         |
| `n` | Show next candidate; escalate to Sonnet if Haiku exhausted    |
| `p` | Show previous candidate                                       |
| `q` | Exit, record nothing                                          |

On `y`: execute the command in the user's shell and stream stdout/stderr.

## Config

`~/.config/pls/config.toml`

```toml
[llm]
api_key = "sk-ant-..."  # or use ANTHROPIC_API_KEY env var

[llm.models]
fast = "claude-haiku-4-5-20251001"
strong = "claude-sonnet-4-5-20250929"
escalation_threshold = 0.7
```

API key resolution: env var `ANTHROPIC_API_KEY` > config file.

## What We Cut

The original spec included significant mechanical complexity that duplicates
what the LLM already does well:

| Original spec feature              | What we do instead                       |
|-------------------------------------|------------------------------------------|
| Bandit-style scoring engine         | LLM reasons over raw history             |
| Intent normalization + synonym maps | LLM handles fuzzy matching               |
| Candidate merging/ranking           | LLM ranks candidates                     |
| Repo fingerprinting + score decay   | LLM sees new context, adapts             |
| Trust zones                         | Removed                                  |
| Safety blocklist                    | LLM risk classification + color coding   |
| Edit mode                           | Copy + edit in terminal                  |
| Pluggable LLM providers             | Anthropic only for v0                    |
| Repo overlay DB                     | Removed                                  |
| Configurable scoring parameters     | Removed                                  |

## Future Considerations (not v0)

- **Exact-match cache**: if telemetry shows repeated identical intents are
  common, add a fast path that skips the LLM call
- **Additional parsers**: Terraform, Kubernetes, Cargo, etc.
- **Pluggable providers**: abstract LLM behind interface if demand arises
