package cmd

import "github.com/spf13/cobra"

var injectCmd = &cobra.Command{
	Use:   "inject <agent> <file>",
	Short: "Send @<file> to agent",
	Long:  `Send a file reference (@<file>) to an agent's tmux session.`,
	Example: `  tagents inject myagent ./context.md
  tagents inject gateway:myagent /tmp/instructions.txt`,
	Args: cobra.ExactArgs(2),
	RunE: runInject,
}

func init() { rootCmd.AddCommand(injectCmd) }

func runInject(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-10
	return nil
}
