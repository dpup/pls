package tui

import "github.com/charmbracelet/lipgloss"

var (
	commandStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	commandBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			MarginLeft(2).
			PaddingLeft(1).
			PaddingRight(1)
	reasonStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).PaddingLeft(2)
	riskSafe      = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	riskModerate  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	riskDangerous = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).PaddingLeft(2).PaddingTop(1)
	keyStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)

	// Verbose context styles.
	sectionHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).PaddingLeft(2)
	labelStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).PaddingLeft(2)
	valueStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	// Tool log styles.
	toolNameStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).PaddingLeft(4)
	toolInputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	toolResultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("242")).PaddingLeft(6)
	toolErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).PaddingLeft(6)
)

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
