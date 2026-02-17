package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
)

// SystemPrompt returns the system prompt that instructs the LLM how to respond.
func SystemPrompt() string {
	return `You are a command-line assistant. Given a user's intent and project context, suggest shell commands that accomplish the task.

Rules:
- Return valid JSON only. No markdown fences, no prose, no explanation outside JSON.
- Return 1-5 candidates ranked by confidence (highest first).
- Allow pipes, chaining, jq, docker exec, psql, and other advanced shell features.
- Prefer commands grounded in the project context (e.g. use the detected package manager, build tool, or scripts).
- Use command history to learn the user's preferences and conventions.
- Avoid repeating commands the user has previously rejected.
- Classify risk for each command:
  - "safe": read-only operations (ls, cat, grep, git status, etc.)
  - "moderate": writes that are reversible (git commit, file edits with backups, etc.)
  - "dangerous": destructive or irreversible operations (rm -rf, DROP TABLE, force push, etc.)

Response JSON schema:
{
  "candidates": [
    {
      "cmd": "the shell command to run",
      "reason": "brief explanation of why this command fits",
      "confidence": 0.95,
      "risk": "safe"
    }
  ]
}`
}

// BuildPrompt constructs the user prompt from intent, context snapshot, and history.
func BuildPrompt(intent string, snap *context.Snapshot, projectHistory []history.Entry, globalHistory []history.Entry) string {
	var b strings.Builder

	// Project context section.
	b.WriteString("## Project Context\n")
	b.WriteString(fmt.Sprintf("repo_root: %s\n", snap.RepoRoot))
	b.WriteString(fmt.Sprintf("cwd_rel: %s\n", snap.CwdRel))
	if len(snap.Results) > 0 {
		b.WriteString("detected_tooling:\n")
		for _, r := range snap.Results {
			data, err := json.Marshal(r.Data)
			if err != nil {
				data = []byte("{}")
			}
			b.WriteString(fmt.Sprintf("  %s: %s\n", r.Name, string(data)))
		}
	}
	b.WriteString("\n")

	// Global history section (recent accepted commands across all repos).
	if len(globalHistory) > 0 {
		b.WriteString("## Recent Global History (accepted commands across repos)\n")
		limit := len(globalHistory)
		if limit > 10 {
			limit = 10
		}
		for _, e := range globalHistory[:limit] {
			b.WriteString(fmt.Sprintf("- intent=%q cmd=%q\n", e.Intent, e.Command))
		}
		b.WriteString("\n")
	}

	// Project history section (commands for this repo+dir, including rejections).
	if len(projectHistory) > 0 {
		b.WriteString("## Project History (commands for this repo+directory)\n")
		limit := len(projectHistory)
		if limit > 20 {
			limit = 20
		}
		for _, e := range projectHistory[:limit] {
			outcome := e.Outcome
			if outcome == history.OutcomeRejected {
				outcome = "rejected (do not repeat)"
			}
			b.WriteString(fmt.Sprintf("- intent=%q cmd=%q outcome=%s\n", e.Intent, e.Command, outcome))
		}
		b.WriteString("\n")
	}

	// User intent section.
	b.WriteString("## User Intent\n")
	b.WriteString(intent)
	b.WriteString("\n")

	return b.String()
}
