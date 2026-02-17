package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pls [intent]",
	Short: "Project-aware natural language shell command router",
	Long:  "Translates natural language into the right shell command for your current project.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		intent := strings.Join(args, " ")
		fmt.Printf("Intent: %s\n", intent)
		// TODO: wire up context -> llm -> tui pipeline
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
