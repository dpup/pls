package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dpup/pls/internal/llm"
)

func makeCandidates(n int) []llm.Candidate {
	candidates := []llm.Candidate{
		{Cmd: "echo hello", Reason: "prints hello", Confidence: 0.9, Risk: "safe"},
		{Cmd: "rm -rf /tmp/junk", Reason: "cleans temp", Confidence: 0.7, Risk: "moderate"},
		{Cmd: "shutdown -h now", Reason: "shuts down", Confidence: 0.5, Risk: "dangerous"},
	}
	if n > len(candidates) {
		n = len(candidates)
	}
	return candidates[:n]
}

func keyMsg(r rune) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestNew(t *testing.T) {
	candidates := makeCandidates(2)
	m := New(candidates)

	if m.index != 0 {
		t.Errorf("expected index 0, got %d", m.index)
	}
	if m.done {
		t.Error("expected done to be false")
	}
	if len(m.candidates) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(m.candidates))
	}
}

func TestUpdate_YKey(t *testing.T) {
	candidates := makeCandidates(1)
	m := New(candidates)

	updated, cmd := m.Update(keyMsg('y'))
	um := updated.(Model)

	if !um.done {
		t.Error("expected done to be true after pressing 'y'")
	}
	if um.result.Action != ActionRun {
		t.Errorf("expected ActionRun (%d), got %d", ActionRun, um.result.Action)
	}
	if um.result.Candidate.Cmd != "echo hello" {
		t.Errorf("expected candidate cmd 'echo hello', got %q", um.result.Candidate.Cmd)
	}
	if cmd == nil {
		t.Error("expected a tea.Quit command, got nil")
	}
}

func TestUpdate_CKey(t *testing.T) {
	candidates := makeCandidates(1)
	m := New(candidates)

	updated, cmd := m.Update(keyMsg('c'))
	um := updated.(Model)

	if !um.done {
		t.Error("expected done to be true after pressing 'c'")
	}
	if um.result.Action != ActionCopy {
		t.Errorf("expected ActionCopy (%d), got %d", ActionCopy, um.result.Action)
	}
	if um.result.Candidate.Cmd != "echo hello" {
		t.Errorf("expected candidate cmd 'echo hello', got %q", um.result.Candidate.Cmd)
	}
	if cmd == nil {
		t.Error("expected a tea.Quit command, got nil")
	}
}

func TestUpdate_QKey(t *testing.T) {
	candidates := makeCandidates(1)
	m := New(candidates)

	updated, cmd := m.Update(keyMsg('q'))
	um := updated.(Model)

	if !um.done {
		t.Error("expected done to be true after pressing 'q'")
	}
	if um.result.Action != ActionQuit {
		t.Errorf("expected ActionQuit (%d), got %d", ActionQuit, um.result.Action)
	}
	if cmd == nil {
		t.Error("expected a tea.Quit command, got nil")
	}
}

func TestUpdate_NextPrev(t *testing.T) {
	candidates := makeCandidates(3)
	m := New(candidates)

	// Start at 0, press 'n' to go to 1.
	updated, _ := m.Update(keyMsg('n'))
	um := updated.(Model)
	if um.index != 1 {
		t.Errorf("after 'n': expected index 1, got %d", um.index)
	}

	// Press 'n' again to go to 2.
	updated, _ = um.Update(keyMsg('n'))
	um = updated.(Model)
	if um.index != 2 {
		t.Errorf("after second 'n': expected index 2, got %d", um.index)
	}

	// Press 'n' at the last index: should stay at 2.
	updated, _ = um.Update(keyMsg('n'))
	um = updated.(Model)
	if um.index != 2 {
		t.Errorf("after 'n' at end: expected index 2, got %d", um.index)
	}

	// Press 'p' to go back to 1.
	updated, _ = um.Update(keyMsg('p'))
	um = updated.(Model)
	if um.index != 1 {
		t.Errorf("after 'p': expected index 1, got %d", um.index)
	}

	// Press 'p' to go back to 0.
	updated, _ = um.Update(keyMsg('p'))
	um = updated.(Model)
	if um.index != 0 {
		t.Errorf("after second 'p': expected index 0, got %d", um.index)
	}

	// Press 'p' at 0: should stay at 0.
	updated, _ = um.Update(keyMsg('p'))
	um = updated.(Model)
	if um.index != 0 {
		t.Errorf("after 'p' at start: expected index 0, got %d", um.index)
	}

	// Model should not be done after navigation.
	if um.done {
		t.Error("expected done to be false after navigation keys")
	}
}

func TestView_ShowsCommand(t *testing.T) {
	candidates := makeCandidates(1)
	m := New(candidates)

	view := m.View()
	if !containsSubstring(view, "echo hello") {
		t.Errorf("expected view to contain 'echo hello', got:\n%s", view)
	}
}

func TestView_DoneReturnsEmpty(t *testing.T) {
	candidates := makeCandidates(1)
	m := New(candidates)

	updated, _ := m.Update(keyMsg('q'))
	um := updated.(Model)

	view := um.View()
	if view != "" {
		t.Errorf("expected empty view after quit, got:\n%s", view)
	}
}

func TestView_ShowsCandidateCounter(t *testing.T) {
	candidates := makeCandidates(3)
	m := New(candidates)

	view := m.View()
	if !containsSubstring(view, "[1/3]") {
		t.Errorf("expected view to contain '[1/3]', got:\n%s", view)
	}

	// Navigate to second candidate.
	updated, _ := m.Update(keyMsg('n'))
	um := updated.(Model)
	view = um.View()
	if !containsSubstring(view, "[2/3]") {
		t.Errorf("expected view to contain '[2/3]', got:\n%s", view)
	}
}

func TestView_NoCandidateCounterForSingle(t *testing.T) {
	candidates := makeCandidates(1)
	m := New(candidates)

	view := m.View()
	if containsSubstring(view, "[1/1]") {
		t.Errorf("expected no candidate counter for single candidate, got:\n%s", view)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
