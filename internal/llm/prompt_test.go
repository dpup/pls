package llm

import (
	"strings"
	"testing"
	"time"

	"github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
)

func TestBuildPrompt_IncludesIntent(t *testing.T) {
	snap := &context.Snapshot{
		RepoRoot: "/home/user/myproject",
		CwdRel:   "src",
		Results:  []context.Result{},
	}

	prompt := BuildPrompt("deploy the app to staging", snap, nil, nil)

	if !strings.Contains(prompt, "deploy the app to staging") {
		t.Errorf("prompt should contain the user intent, got:\n%s", prompt)
	}
}

func TestBuildPrompt_IncludesContext(t *testing.T) {
	snap := &context.Snapshot{
		RepoRoot: "/home/user/myproject",
		CwdRel:   "services/api",
		Results: []context.Result{
			{
				Name: "node",
				Data: map[string]any{
					"package_manager": "pnpm",
					"scripts":        []any{"build", "test", "lint"},
				},
			},
		},
	}

	prompt := BuildPrompt("run tests", snap, nil, nil)

	if !strings.Contains(prompt, "pnpm") {
		t.Errorf("prompt should contain the package manager 'pnpm', got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "services/api") {
		t.Errorf("prompt should contain cwd_rel 'services/api', got:\n%s", prompt)
	}
}

func TestBuildPrompt_IncludesHistory(t *testing.T) {
	snap := &context.Snapshot{
		RepoRoot: "/home/user/myproject",
		CwdRel:   ".",
		Results:  []context.Result{},
	}

	projectHistory := []history.Entry{
		{
			ID:        1,
			Intent:    "run tests",
			Command:   "go test ./...",
			Outcome:   history.OutcomeAccepted,
			CreatedAt: time.Now(),
		},
		{
			ID:        2,
			Intent:    "run tests",
			Command:   "make test",
			Outcome:   history.OutcomeRejected,
			CreatedAt: time.Now(),
		},
	}

	globalHistory := []history.Entry{
		{
			ID:        10,
			Intent:    "check disk usage",
			Command:   "df -h",
			Outcome:   history.OutcomeAccepted,
			CreatedAt: time.Now(),
		},
	}

	prompt := BuildPrompt("run tests", snap, projectHistory, globalHistory)

	if !strings.Contains(prompt, "go test ./...") {
		t.Errorf("prompt should contain project history command 'go test ./...', got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "rejected") {
		t.Errorf("prompt should contain rejection info for rejected commands, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "df -h") {
		t.Errorf("prompt should contain global history command 'df -h', got:\n%s", prompt)
	}
}

func TestSystemPrompt_IsReasonable(t *testing.T) {
	sp := SystemPrompt()

	if !strings.Contains(sp, "candidates") {
		t.Errorf("system prompt should mention 'candidates', got:\n%s", sp)
	}
	if !strings.Contains(sp, "risk") {
		t.Errorf("system prompt should mention 'risk', got:\n%s", sp)
	}
}
