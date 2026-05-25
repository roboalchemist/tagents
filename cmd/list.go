package cmd

import "github.com/spf13/cobra"

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agent sessions",
	Long:  `List all agent sessions. Columns: machine (multi-scope), session name, runtime, status, cwd, preview.`,
	Example: `  tagents list
  tagents list --json
  tagents list --machine gateway
  tagents list --all-machines`,
	RunE: runList,
}

func init() { rootCmd.AddCommand(listCmd) }

func runList(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-8
	return nil
}
