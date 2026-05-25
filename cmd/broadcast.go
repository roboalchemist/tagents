package cmd

import "github.com/spf13/cobra"

var broadcastCmd = &cobra.Command{
	Use:   "broadcast <message>",
	Short: "Send a message to all idle agents",
	Long: `Send a message to all idle agents across all discovered machines.

Examples:
  tagents broadcast "daily standup: what did you accomplish?"
  tagents broadcast --machine gateway "please wrap up"`,
	Args: cobra.ExactArgs(1),
	RunE: runBroadcast,
}

func init() { rootCmd.AddCommand(broadcastCmd) }

func runBroadcast(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-10
	return nil
}
