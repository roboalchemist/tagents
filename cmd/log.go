package cmd

import "github.com/spf13/cobra"

var logCmd = &cobra.Command{
	Use:   "log <agent> [lines]",
	Short: "Read agent session log",
	Long:  `Read the agent's session transcript (Claude Code JSONL or Codex format).`,
	Example: `  tagents log myagent
  tagents log myagent 200`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runLog,
}

func init() { rootCmd.AddCommand(logCmd) }

func runLog(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-9
	return nil
}
