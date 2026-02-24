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

// runOptions holds CLI flag values passed from the command to the run function.
type runOptions struct {
	printJSON bool
	verbose   bool
	explain   bool
}

// newRootCmd creates a fresh *cobra.Command with flags scoped to the closure,
// eliminating package-level mutable state. Each call returns an isolated instance.
func newRootCmd() *cobra.Command {
	var opts runOptions

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
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(strings.Join(args, " "), opts)
		},
	}

	cmd.Flags().BoolVar(&opts.printJSON, "json", false, "Print candidates as JSON instead of interactive TUI")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "v", false, "Print context sent to LLM before showing results")
	cmd.Flags().BoolVar(&opts.explain, "explain", false, "Print the prompt that would be sent to the LLM, then exit (no API call)")

	return cmd
}

func run(intent string, opts runOptions) error {
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

	// Query history (best-effort).
	repoID, projectHistory, globalHistory := queryHistory(store, snap, opts.verbose)

	// --explain: print the full prompt and exit without calling the API.
	if opts.explain {
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
	if opts.verbose {
		fmt.Fprint(os.Stderr, tui.PrintContext(*snap, projectHistory, globalHistory))
	}

	// Generate candidates
	resp, err := generateCandidates(cfg, snap, intent, projectHistory, globalHistory, opts.verbose)
	if err != nil {
		return err
	}

	if len(resp.Candidates) == 0 {
		fmt.Println("No command candidates generated.")
		return nil
	}

	// Non-interactive mode
	if opts.printJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	// Run TUI, record outcome, and execute.
	return runInteractive(resp, store, repoID, snap.CwdRel, intent, opts.verbose)
}

// queryHistory loads history from the store, logging warnings under --verbose.
func queryHistory(store *history.Store, snap *plsctx.Snapshot, verbose bool) (int64, []history.Entry, []history.Entry) {
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
	return repoID, projectHistory, globalHistory
}

// generateCandidates calls the LLM and returns the response.
func generateCandidates(cfg *config.Config, snap *plsctx.Snapshot, intent string, projectHistory, globalHistory []history.Entry, verbose bool) (*llm.Response, error) {
	client := llm.NewClient(cfg)
	resp, err := client.Generate(*snap, intent, projectHistory, globalHistory)
	if err != nil {
		return nil, fmt.Errorf("generating commands: %w", err)
	}
	if verbose {
		fmt.Fprint(os.Stderr, tui.PrintToolLog(resp.Rounds))
	}
	return resp, nil
}

// runInteractive presents the TUI, records the outcome, and executes the chosen action.
func runInteractive(resp *llm.Response, store *history.Store, repoID int64, cwdRel, intent string, verbose bool) error {
	result, err := tui.RunTUI(resp.Candidates)
	if err != nil {
		return fmt.Errorf("TUI: %w", err)
	}

	switch result.Action {
	case tui.ActionRun:
		if err := store.Record(repoID, cwdRel, intent, result.Candidate.Cmd, history.OutcomeAccepted); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: history record: %v\n", err)
		}
	case tui.ActionCopy:
		if err := store.Record(repoID, cwdRel, intent, result.Candidate.Cmd, history.OutcomeCopied); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: history record: %v\n", err)
		}
	}

	return tui.Execute(result)
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
