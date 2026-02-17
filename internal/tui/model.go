package tui

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dpup/pls/internal/exec"
	"github.com/dpup/pls/internal/llm"
)

// Action represents the user's chosen action from the TUI.
type Action int

const (
	ActionNone Action = iota
	ActionRun
	ActionCopy
	ActionQuit
)

// Result holds the chosen action and the selected candidate.
type Result struct {
	Action    Action
	Candidate llm.Candidate
}

// Model is the bubbletea model for the interactive TUI.
type Model struct {
	candidates []llm.Candidate
	index      int
	result     Result
	done       bool
}

// New creates a new TUI Model from a list of candidates.
func New(candidates []llm.Candidate) Model {
	return Model{
		candidates: candidates,
		index:      0,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
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

// View implements tea.Model.
func (m Model) View() string {
	if m.done {
		return ""
	}

	c := m.candidates[m.index]

	// Command line.
	s := commandBoxStyle.Render(commandStyle.Render(c.Cmd)) + "\n"

	// Reason line.
	s += "\n"
	s += reasonStyle.Render("Reason: "+c.Reason) + "\n"

	// Risk line with optional candidate counter.
	riskLine := "Risk:   " + riskLabel(c.Risk)
	if len(m.candidates) > 1 {
		riskLine += fmt.Sprintf("        [%d/%d]", m.index+1, len(m.candidates))
	}
	s += reasonStyle.Render(riskLine) + "\n"

	// Help bar.
	help := keyStyle.Render("[y]") + " run  " +
		keyStyle.Render("[c]") + " copy  " +
		keyStyle.Render("[n]") + " next  " +
		keyStyle.Render("[p]") + " prev  " +
		keyStyle.Render("[q]") + " quit"
	s += helpStyle.Render(help) + "\n"

	return s
}

// RunTUI creates a bubbletea program, runs it, and returns the result.
func RunTUI(candidates []llm.Candidate) (*Result, error) {
	m := New(candidates)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}
	fm := finalModel.(Model)
	return &fm.result, nil
}

// Execute handles the action from the TUI result.
func Execute(result *Result) error {
	if result == nil {
		return nil
	}

	switch result.Action {
	case ActionRun:
		fmt.Fprintln(os.Stderr)
		return exec.Run(result.Candidate.Cmd, os.Stdout)
	case ActionCopy:
		err := clipboard.WriteAll(result.Candidate.Cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not copy to clipboard: %v\nCommand: %s\n", err, result.Candidate.Cmd)
		} else {
			fmt.Fprintf(os.Stderr, "Copied to clipboard: %s\n", result.Candidate.Cmd)
		}
		return nil
	case ActionQuit:
		return nil
	default:
		return nil
	}
}
