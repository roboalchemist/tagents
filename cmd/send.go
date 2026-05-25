package cmd

import "github.com/spf13/cobra"

var sendCmd = &cobra.Command{
	Use:   "send <agent> <message>",
	Short: "Send a message to an agent",
	Long: `Send a message to an agent's tmux session via send-keys.

Examples:
  tagents send myagent "please continue"
  tagents send gateway:myagent "what is your status?"`,
	Args: cobra.ExactArgs(2),
	RunE: runSend,
}

func init() { rootCmd.AddCommand(sendCmd) }

func runSend(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-10
	return nil
}
