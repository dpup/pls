package tui

import (
	"strings"
	"testing"

	plsctx "github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
	"github.com/dpup/pls/internal/llm"
)

func TestPrintContext_RendersDetected(t *testing.T) {
	snap := plsctx.Snapshot{
		RepoRoot: "/home/user/project",
		CwdRel:   ".",
		Results: []plsctx.Result{
			{
				Name: "git",
				Data: map[string]any{
					"branch": "main",
					"dirty":  false,
				},
			},
			{
				Name: "node",
				Data: map[string]any{
					"version": "18.0.0",
				},
			},
		},
	}

	output := PrintContext(snap, nil, nil)

	if !strings.Contains(output, "Context") {
		t.Errorf("expected output to contain 'Context', got:\n%s", output)
	}
	if !strings.Contains(output, "Detected") {
		t.Errorf("expected output to contain 'Detected', got:\n%s", output)
	}
	if !strings.Contains(output, "git") {
		t.Errorf("expected output to contain parser name 'git', got:\n%s", output)
	}
	if !strings.Contains(output, "node") {
		t.Errorf("expected output to contain parser name 'node', got:\n%s", output)
	}
	if !strings.Contains(output, "branch") {
		t.Errorf("expected output to contain data key 'branch', got:\n%s", output)
	}
	if !strings.Contains(output, "main") {
		t.Errorf("expected output to contain data value 'main', got:\n%s", output)
	}
	if !strings.Contains(output, "version") {
		t.Errorf("expected output to contain data key 'version', got:\n%s", output)
	}
	if !strings.Contains(output, "18.0.0") {
		t.Errorf("expected output to contain data value '18.0.0', got:\n%s", output)
	}
}

func TestPrintContext_WithHistory(t *testing.T) {
	snap := plsctx.Snapshot{
		Results: []plsctx.Result{},
	}
	projectHistory := []history.Entry{
		{Intent: "list files", Command: "ls -la", Outcome: "accepted"},
	}
	globalHistory := []history.Entry{
		{Intent: "disk usage", Command: "df -h", Outcome: "accepted"},
	}

	output := PrintContext(snap, projectHistory, globalHistory)

	if !strings.Contains(output, "History (project)") {
		t.Errorf("expected output to contain 'History (project)', got:\n%s", output)
	}
	if !strings.Contains(output, "list files") {
		t.Errorf("expected output to contain 'list files', got:\n%s", output)
	}
	if !strings.Contains(output, "ls -la") {
		t.Errorf("expected output to contain 'ls -la', got:\n%s", output)
	}
	if !strings.Contains(output, "History (global)") {
		t.Errorf("expected output to contain 'History (global)', got:\n%s", output)
	}
	if !strings.Contains(output, "disk usage") {
		t.Errorf("expected output to contain 'disk usage', got:\n%s", output)
	}
	if !strings.Contains(output, "df -h") {
		t.Errorf("expected output to contain 'df -h', got:\n%s", output)
	}
}

func TestPrintToolLog_Empty(t *testing.T) {
	result := PrintToolLog(nil)
	if result != "" {
		t.Errorf("expected empty string for nil rounds, got:\n%s", result)
	}

	result = PrintToolLog([]llm.ToolRound{})
	if result != "" {
		t.Errorf("expected empty string for empty rounds, got:\n%s", result)
	}
}

func TestPrintToolLog_WithRounds(t *testing.T) {
	rounds := []llm.ToolRound{
		{
			Calls: []llm.ToolCall{
				{
					Name:    "read_file",
					Input:   map[string]any{"path": "/etc/hosts"},
					Result:  "127.0.0.1 localhost",
					IsError: false,
				},
			},
		},
		{
			Calls: []llm.ToolCall{
				{
					Name:    "list_dir",
					Input:   map[string]any{"dir": "/tmp"},
					Result:  "file1.txt\nfile2.txt\nfile3.txt",
					IsError: false,
				},
				{
					Name:    "run_cmd",
					Input:   map[string]any{"cmd": "whoami"},
					Result:  "",
					IsError: true,
				},
			},
		},
	}

	output := PrintToolLog(rounds)

	if !strings.Contains(output, "LLM Tool Use") {
		t.Errorf("expected output to contain 'LLM Tool Use', got:\n%s", output)
	}
	if !strings.Contains(output, "2 rounds") {
		t.Errorf("expected output to contain '2 rounds', got:\n%s", output)
	}
	if !strings.Contains(output, "Round 1") {
		t.Errorf("expected output to contain 'Round 1', got:\n%s", output)
	}
	if !strings.Contains(output, "Round 2") {
		t.Errorf("expected output to contain 'Round 2', got:\n%s", output)
	}
	if !strings.Contains(output, "read_file") {
		t.Errorf("expected output to contain tool name 'read_file', got:\n%s", output)
	}
	if !strings.Contains(output, "list_dir") {
		t.Errorf("expected output to contain tool name 'list_dir', got:\n%s", output)
	}
	if !strings.Contains(output, "run_cmd") {
		t.Errorf("expected output to contain tool name 'run_cmd', got:\n%s", output)
	}
	if !strings.Contains(output, "127.0.0.1 localhost") {
		t.Errorf("expected output to contain result '127.0.0.1 localhost', got:\n%s", output)
	}
	// The multiline result should show line count.
	if !strings.Contains(output, "3 lines") {
		t.Errorf("expected output to contain '3 lines' for multiline result, got:\n%s", output)
	}
	// The error result should show "error:".
	if !strings.Contains(output, "error:") {
		t.Errorf("expected output to contain 'error:' for error result, got:\n%s", output)
	}
}

func TestPrintToolLog_SingleRound(t *testing.T) {
	rounds := []llm.ToolRound{
		{
			Calls: []llm.ToolCall{
				{
					Name:    "test_tool",
					Input:   map[string]any{},
					Result:  "ok",
					IsError: false,
				},
			},
		},
	}

	output := PrintToolLog(rounds)

	// Single round should not have "rounds" (plural).
	if !strings.Contains(output, "1 round)") {
		t.Errorf("expected output to contain '1 round)' (singular), got:\n%s", output)
	}
	if strings.Contains(output, "1 rounds)") {
		t.Errorf("expected no '1 rounds)' (wrong plural), got:\n%s", output)
	}
}

func TestFormatValue_MapStringString(t *testing.T) {
	m := map[string]any{
		"targets": map[string]string{
			"test":  "Run all tests",
			"build": "Build the binary",
			"lint":  "",
		},
	}

	// Simulate what PrintContext does for a result.
	output := formatValue(m["targets"])

	// Should show "target: description" for targets with descriptions.
	if !strings.Contains(output, "test: Run all tests") {
		t.Errorf("expected 'test: Run all tests' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "build: Build the binary") {
		t.Errorf("expected 'build: Build the binary' in output, got:\n%s", output)
	}
	// Target without description should just show the name.
	if !strings.Contains(output, "lint") {
		t.Errorf("expected 'lint' in output, got:\n%s", output)
	}
	// Should NOT contain Go map dump format.
	if strings.Contains(output, "map[") {
		t.Errorf("should not contain Go map dump 'map[', got:\n%s", output)
	}
}

func TestFormatPrompt(t *testing.T) {
	prompt := "You are a helpful CLI assistant.\nPlease suggest a command."

	output := FormatPrompt(prompt)

	if !strings.Contains(output, "Prompt") {
		t.Errorf("expected output to contain 'Prompt', got:\n%s", output)
	}
	if !strings.Contains(output, "You are a helpful CLI assistant.") {
		t.Errorf("expected output to contain prompt text, got:\n%s", output)
	}
	if !strings.Contains(output, "Please suggest a command.") {
		t.Errorf("expected output to contain prompt second line, got:\n%s", output)
	}
}
