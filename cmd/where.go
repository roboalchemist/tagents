package cmd

import "github.com/spf13/cobra"

var whereCmd = &cobra.Command{
	Use:   "where <agent>",
	Short: "Get agent's current working directory",
	Long:  `Print the current working directory of the agent's tmux pane.`,
	Example: `  tagents where myagent
  tagents where gateway:myagent`,
	Args: cobra.ExactArgs(1),
	RunE: runWhere,
}

func init() { rootCmd.AddCommand(whereCmd) }

func runWhere(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-9
	return nil
}
