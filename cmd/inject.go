package cmd

import (
	"fmt"
	"os"

	"github.com/roboalchemist/tagents/pkg/tmux"
	"github.com/spf13/cobra"
)

var flagInjectDryRun bool

var injectCmd = &cobra.Command{
	Use:   "inject <agent> <file>",
	Short: "Send @<file> to an agent",
	Long: `Send '@<file>' to an agent — points it at a goal file.

The file path must exist. The message sent is literally '@<file>'.

Examples:
  tagents inject my-agent /tmp/goal.md
  tagents inject worker ~/tasks/next.md`,
	Example: `  tagents inject my-agent /tmp/goal.md
  tagents inject worker ~/tasks/next.md`,
	Args: cobra.ExactArgs(2),
	RunE: runInject,
}

func init() {
	injectCmd.Flags().BoolVar(&flagInjectDryRun, "dry-run", false, "Print what would be injected without sending")
	rootCmd.AddCommand(injectCmd)
}

func runInject(cmd *cobra.Command, args []string) error {
	query := args[0]
	filePath := args[1]

	// Expand ~ in file path
	if len(filePath) >= 2 && filePath[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		filePath = home + filePath[1:]
	}

	if flagInjectDryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] Would inject %s to %s\n", filePath, query)
		return nil
	}

	// Validate file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", filePath)
	}

	s, err := findSession(query)
	if err != nil {
		return err
	}

	message := "@" + filePath
	client := tmux.NewLocalClient()
	if err := client.SendKeys(s.Name, message); err != nil {
		return fmt.Errorf("injecting to %q: %w", s.Name, err)
	}

	fmt.Printf("Injected %s to %s\n", filePath, s.Name)
	return nil
}
