package cmd

import "github.com/spf13/cobra"

var readCmd = &cobra.Command{
	Use:   "read <agent> [lines]",
	Short: "Read last N lines of agent's tmux pane output",
	Long: `Read the last N lines of an agent's tmux pane output (default 50).

Examples:
  tagents read myagent
  tagents read myagent 100
  tagents read gateway:myagent`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRead,
}

func init() { rootCmd.AddCommand(readCmd) }

func runRead(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-9
	return nil
}
