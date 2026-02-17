package tui

import "github.com/charmbracelet/lipgloss"

var (
	commandStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).PaddingLeft(2)
	reasonStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).PaddingLeft(2)
	riskSafe     = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	riskModerate = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	riskDangerous = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingLeft(2).PaddingTop(1)
	keyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
)

// riskStyle returns the appropriate lipgloss style for the given risk level.
func riskStyle(risk string) lipgloss.Style {
	switch risk {
	case "safe":
		return riskSafe
	case "moderate":
		return riskModerate
	case "dangerous":
		return riskDangerous
	default:
		return riskSafe
	}
}

// riskLabel returns a color-coded risk label string for display.
func riskLabel(risk string) string {
	switch risk {
	case "safe":
		return riskSafe.Render("\u25a0 safe")
	case "moderate":
		return riskModerate.Render("\u25a0 moderate")
	case "dangerous":
		return riskDangerous.Render("\u25a0 dangerous")
	default:
		return riskSafe.Render("\u25a0 safe")
	}
}
