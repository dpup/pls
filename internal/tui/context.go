package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	plsctx "github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
	"github.com/dpup/pls/internal/llm"
)

var (
	sectionRule = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
)

// PrintContext renders the LLM context (detected environment, project history,
// global history) to stderr so that --verbose output doesn't interfere with
// --json on stdout.
func PrintContext(snap plsctx.Snapshot, projectHistory, globalHistory []history.Entry) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(sectionHeaderStyle.Render("Context"))
	b.WriteString("\n")

	// Detected context.
	header := sectionRule.Render("── Detected " + strings.Repeat("─", 26))
	b.WriteString(labelStyle.Render(header))
	b.WriteString("\n")

	for _, r := range snap.Results {
		b.WriteString(labelStyle.Render(fmt.Sprintf("%-10s", r.Name)))
		b.WriteString("\n")
		for k, v := range r.Data {
			b.WriteString(labelStyle.Render(fmt.Sprintf("  %-16s", k)))
			b.WriteString(valueStyle.Render(formatValue(v)))
			b.WriteString("\n")
		}
	}

	// Project history.
	if len(projectHistory) > 0 {
		b.WriteString("\n")
		header = sectionRule.Render("── History (project) " + strings.Repeat("─", 17))
		b.WriteString(labelStyle.Render(header))
		b.WriteString("\n")
		for _, e := range projectHistory {
			line := labelStyle.Render(fmt.Sprintf("%q → %s", e.Intent, e.Command)) +
				valueStyle.Render(fmt.Sprintf("  [%s]", e.Outcome))
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Global history.
	if len(globalHistory) > 0 {
		b.WriteString("\n")
		header = sectionRule.Render("── History (global) " + strings.Repeat("─", 18))
		b.WriteString(labelStyle.Render(header))
		b.WriteString("\n")
		for _, e := range globalHistory {
			line := labelStyle.Render(fmt.Sprintf("%q → %s", e.Intent, e.Command)) +
				valueStyle.Render(fmt.Sprintf("  [%s]", e.Outcome))
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

// FormatPrompt renders the full LLM prompt for --explain output.
func FormatPrompt(prompt string) string {
	var b strings.Builder
	header := sectionRule.Render("── Prompt " + strings.Repeat("─", 29))
	b.WriteString(labelStyle.Render(header))
	b.WriteString("\n")
	for _, line := range strings.Split(prompt, "\n") {
		b.WriteString(labelStyle.Render(line))
		b.WriteString("\n")
	}
	return b.String()
}

// PrintToolLog renders the tool-use rounds from the LLM generation loop.
func PrintToolLog(rounds []llm.ToolRound) string {
	if len(rounds) == 0 {
		return ""
	}

	var b strings.Builder

	header := sectionRule.Render(fmt.Sprintf("── LLM Tool Use (%d round%s) ", len(rounds), plural(len(rounds))) + strings.Repeat("─", 16))
	b.WriteString(labelStyle.Render(header))
	b.WriteString("\n")

	for i, round := range rounds {
		b.WriteString(labelStyle.Render(fmt.Sprintf("Round %d", i+1)))
		b.WriteString("\n")

		for _, call := range round.Calls {
			// Tool name and input args.
			args := formatToolInput(call.Input)
			b.WriteString(toolNameStyle.Render("-> "+call.Name) + toolInputStyle.Render("("+args+")"))
			b.WriteString("\n")

			// Result summary (truncated).
			if call.IsError {
				b.WriteString(toolErrorStyle.Render("error: " + truncate(call.Result, 120)))
			} else {
				lines := strings.Count(call.Result, "\n") + 1
				preview := firstLine(call.Result)
				if lines > 1 {
					b.WriteString(toolResultStyle.Render(fmt.Sprintf("<- %s ... (%d lines)", truncate(preview, 60), lines)))
				} else {
					b.WriteString(toolResultStyle.Render("<- " + truncate(call.Result, 100)))
				}
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

func formatValue(v any) string {
	switch val := v.(type) {
	case []any:
		items := make([]string, len(val))
		for i, item := range val {
			items[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(items, ", ")
	case []string:
		return strings.Join(val, ", ")
	case map[string]string:
		items := make([]string, 0, len(val))
		for k, desc := range val {
			if desc != "" {
				items = append(items, k+": "+desc)
			} else {
				items = append(items, k)
			}
		}
		return strings.Join(items, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatToolInput(input map[string]any) string {
	var parts []string
	for k, v := range input {
		parts = append(parts, fmt.Sprintf("%s: %v", k, v))
	}
	return strings.Join(parts, ", ")
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
