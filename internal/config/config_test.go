package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FromFile(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(`
[llm]
api_key = "sk-ant-test"

[llm.models]
fast = "claude-haiku-4-5-20251001"
strong = "claude-sonnet-4-5-20250929"
escalation_threshold = 0.8
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.APIKey != "sk-ant-test" {
		t.Errorf("unexpected api_key: %q", cfg.LLM.APIKey)
	}
	if cfg.LLM.Models.EscalationThreshold != 0.8 {
		t.Errorf("unexpected threshold: %v", cfg.LLM.Models.EscalationThreshold)
	}
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := LoadFrom("/nonexistent/path")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.Models.Fast != "claude-haiku-4-5-20251001" {
		t.Errorf("unexpected default fast model: %q", cfg.LLM.Models.Fast)
	}
	if cfg.LLM.Models.EscalationThreshold != 0.7 {
		t.Errorf("unexpected default threshold: %v", cfg.LLM.Models.EscalationThreshold)
	}
}

func TestLoad_EnvOverridesKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-from-env")
	cfg, err := LoadFrom("/nonexistent/path")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LLM.APIKey != "sk-from-env" {
		t.Errorf("expected env override, got %q", cfg.LLM.APIKey)
	}
}
