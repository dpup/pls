package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dpup/pls/internal/config"
	plsctx "github.com/dpup/pls/internal/context"
	plsexec "github.com/dpup/pls/internal/exec"
	"github.com/dpup/pls/internal/history"
	"github.com/dpup/pls/internal/llm"
	"github.com/dpup/pls/internal/tui"
)

// newRootCmd creates a fresh *cobra.Command with flags scoped to the closure,
// eliminating package-level mutable state. Each call returns an isolated instance.
func newRootCmd() *cobra.Command {
	var printJSON, verbose, explain bool

	cmd := &cobra.Command{
		Use:   "pls [intent]",
		Short: "Project-aware natural language shell command router",
		Long:  "Translates natural language into the right shell command for your current project.",
		Example: `  pls "run tests"
  pls "deploy to staging"
  pls --json "run the linter"
  pls --explain "run tests for history"`,
		Args:          cobra.MinimumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.Flags().BoolVar(&printJSON, "json", false, "Print candidates as JSON instead of interactive TUI")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Print context sent to LLM before showing results")
	cmd.Flags().BoolVar(&explain, "explain", false, "Print the prompt that would be sent to the LLM, then exit (no API call)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		intent := strings.Join(args, " ")

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Collect context
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		snap, err := plsctx.Collect(cwd, plsctx.DefaultParsers())
		if err != nil {
			return fmt.Errorf("collecting context: %w", err)
		}

		// Open history store
		store, err := history.Open(historyDBPath())
		if err != nil {
			return fmt.Errorf("opening history: %w", err)
		}
		defer store.Close() //nolint:errcheck

		// Query history (best-effort: log errors under --verbose).
		repoID, err := store.EnsureRepo(snap.RepoRoot)
		if err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: history: %v\n", err)
		}
		projectHistory, err := store.ProjectHistory(repoID, snap.CwdRel, 20)
		if err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: history: %v\n", err)
		}
		globalHistory, err := store.RecentGlobal(10)
		if err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: history: %v\n", err)
		}

		// --explain: print the full prompt and exit without calling the API.
		if explain {
			fmt.Fprint(os.Stderr, tui.PrintContext(*snap, projectHistory, globalHistory))
			prompt := llm.BuildPrompt(intent, snap, projectHistory, globalHistory)
			fmt.Fprintln(os.Stderr, tui.FormatPrompt(prompt))
			return nil
		}

		// Check API key after --explain (explain doesn't need it).
		if cfg.LLM.APIKey == "" {
			return fmt.Errorf("no API key configured. Set ANTHROPIC_API_KEY or add api_key to ~/.config/pls/config.toml")
		}

		// Print verbose context before LLM call so the user sees it while waiting.
		if verbose {
			fmt.Fprint(os.Stderr, tui.PrintContext(*snap, projectHistory, globalHistory))
		}

		// Generate candidates
		client := llm.NewClient(cfg)
		resp, err := client.Generate(*snap, intent, projectHistory, globalHistory)
		if err != nil {
			return fmt.Errorf("generating commands: %w", err)
		}

		// Print tool-use log if verbose and tools were used.
		if verbose {
			fmt.Fprint(os.Stderr, tui.PrintToolLog(resp.Rounds))
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
			if err := store.Record(repoID, snap.CwdRel, intent, result.Candidate.Cmd, history.OutcomeAccepted); err != nil && verbose {
				fmt.Fprintf(os.Stderr, "Warning: history record: %v\n", err)
			}
		case tui.ActionCopy:
			if err := store.Record(repoID, snap.CwdRel, intent, result.Candidate.Cmd, history.OutcomeCopied); err != nil && verbose {
				fmt.Fprintf(os.Stderr, "Warning: history record: %v\n", err)
			}
		}

		// Execute the action
		return tui.Execute(result)
	}

	return cmd
}

// Execute is the entry point for the CLI, called from main.
func Execute(version string) {
	cmd := newRootCmd()
	cmd.Version = version
	if err := cmd.Execute(); err != nil {
		// If the executed command failed, forward its exit code silently.
		var exitErr *plsexec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		// For pls's own errors, print the error message.
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func historyDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "pls", "history.db")
	}
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "pls", "history.db")
}
