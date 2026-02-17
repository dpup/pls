package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	plsctx "github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
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
		parts := []string{}
		for k, v := range r.Data {
			parts = append(parts, fmt.Sprintf("%s: %v", k, v))
		}
		line := labelStyle.Render(fmt.Sprintf("%-10s", r.Name)) + valueStyle.Render(strings.Join(parts, ", "))
		b.WriteString(line)
		b.WriteString("\n")
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
