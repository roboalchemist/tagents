package cmd

import (
	"fmt"
	"os"

	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/roboalchemist/tagents/pkg/tmux"
	"github.com/spf13/cobra"
)

var flagSendForce bool

var sendCmd = &cobra.Command{
	Use:   "send <agent> <message>",
	Short: "Send a message to an agent",
	Long: `Send a message to an agent as if typed + Enter in its tmux pane.

Agent name is fuzzy-matched. If the agent is busy, asks for confirmation unless --force is given.

Examples:
  tagents send my-agent "please continue"
  tagents send my-agent "what is the status?" --force
  tagents send gateway:worker "stop and report"`,
	Example: `  tagents send my-agent "please continue"
  tagents send my-agent "check status" --force`,
	Args: cobra.ExactArgs(2),
	RunE: runSend,
}

func init() {
	sendCmd.Flags().BoolVarP(&flagSendForce, "force", "f", false, "Send even if agent is busy")
	rootCmd.AddCommand(sendCmd)
}

func runSend(cmd *cobra.Command, args []string) error {
	query := args[0]
	message := args[1]

	s, err := findSession(query)
	if err != nil {
		return err
	}

	// Warn if agent is busy
	if s.Status == runtime.Busy && !flagSendForce {
		fmt.Fprintf(os.Stderr, "Warning: agent %q is busy. Use --force to send anyway.\n", s.Name)
		return fmt.Errorf("agent is busy (use --force to override)")
	}

	client := tmux.NewLocalClient()
	if err := client.SendKeys(s.Name, message); err != nil {
		return fmt.Errorf("sending to %q: %w", s.Name, err)
	}

	fmt.Printf("Sent to %s\n", s.Name)
	return nil
}
