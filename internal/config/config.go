package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

type Config struct {
	LLM LLMConfig `toml:"llm"`
}

type LLMConfig struct {
	APIKey string       `toml:"api_key"`
	Models ModelsConfig `toml:"models"`
}

type ModelsConfig struct {
	Fast                string  `toml:"fast"`
	Strong              string  `toml:"strong"`
	EscalationThreshold float64 `toml:"escalation_threshold"`
}

func defaults() Config {
	return Config{
		LLM: LLMConfig{
			Models: ModelsConfig{
				Fast:                "claude-haiku-4-5-20251001",
				Strong:              "claude-sonnet-4-5-20250929",
				EscalationThreshold: 0.7,
			},
		},
	}
}

func Load() (*Config, error) {
	return LoadFrom(defaultPath())
}

func LoadFrom(path string) (*Config, error) {
	cfg := defaults()

	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return nil, err
		}
	}

	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.LLM.APIKey = key
	}

	return &cfg, nil
}

func defaultPath() string {
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "pls", "config.toml")
	}
	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir == "" {
		home, _ := os.UserHomeDir()
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "pls", "config.toml")
}
