package cmd

import "github.com/spf13/cobra"

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Fleet summary counts",
	Long:    `Display fleet summary counts: total, busy, idle, dead agents.`,
	Example: `  tagents status
  tagents status --json`,
	RunE: runStatus,
}

func init() { rootCmd.AddCommand(statusCmd) }

func runStatus(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-8
	return nil
}
