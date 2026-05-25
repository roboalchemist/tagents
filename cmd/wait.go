package cmd

import "github.com/spf13/cobra"

var waitCmd = &cobra.Command{
	Use:   "wait <agent> [timeout]",
	Short: "Wait for agent to go idle",
	Long: `Block until the agent's status becomes idle (default timeout: 60s).

Examples:
  tagents wait myagent
  tagents wait myagent 120s
  tagents wait gateway:myagent 5m`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runWait,
}

func init() { rootCmd.AddCommand(waitCmd) }

func runWait(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-10
	return nil
}
