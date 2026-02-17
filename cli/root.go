package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dpup/pls/internal/config"
	plsctx "github.com/dpup/pls/internal/context"
	"github.com/dpup/pls/internal/history"
	"github.com/dpup/pls/internal/llm"
	"github.com/dpup/pls/internal/tui"
)

var printJSON bool

var rootCmd = &cobra.Command{
	Use:   "pls [intent]",
	Short: "Project-aware natural language shell command router",
	Long:  "Translates natural language into the right shell command for your current project.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  run,
}

func init() {
	rootCmd.Flags().BoolVar(&printJSON, "json", false, "Print candidates as JSON instead of interactive TUI")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	intent := strings.Join(args, " ")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if cfg.LLM.APIKey == "" {
		return fmt.Errorf("no API key configured. Set ANTHROPIC_API_KEY or add api_key to ~/.config/pls/config.toml")
	}

	// Collect context
	cwd, _ := os.Getwd()
	snap, err := plsctx.Collect(cwd, plsctx.DefaultParsers())
	if err != nil {
		return fmt.Errorf("collecting context: %w", err)
	}

	// Open history store
	store, err := history.Open(historyDBPath())
	if err != nil {
		return fmt.Errorf("opening history: %w", err)
	}
	defer store.Close()

	// Query history
	repoID, _ := store.EnsureRepo(snap.RepoRoot)
	projectHistory, _ := store.ProjectHistory(repoID, snap.CwdRel, 20)
	globalHistory, _ := store.RecentGlobal(10)

	// Generate candidates
	client := llm.NewClient(cfg)
	resp, err := client.Generate(*snap, intent, projectHistory, globalHistory)
	if err != nil {
		return fmt.Errorf("generating commands: %w", err)
	}
	if len(resp.Candidates) == 0 {
		fmt.Println("No command candidates generated.")
		return nil
	}

	// Non-interactive mode
	if printJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	// Run TUI
	result, err := tui.RunTUI(resp.Candidates)
	if err != nil {
		return fmt.Errorf("TUI: %w", err)
	}

	// Record outcome
	switch result.Action {
	case tui.ActionRun:
		store.Record(repoID, snap.CwdRel, intent, result.Candidate.Cmd, history.OutcomeAccepted)
	case tui.ActionCopy:
		store.Record(repoID, snap.CwdRel, intent, result.Candidate.Cmd, history.OutcomeCopied)
	}

	// Execute the action
	return tui.Execute(result)
}

func historyDBPath() string {
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "pls", "history.db")
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "pls", "history.db")
}
