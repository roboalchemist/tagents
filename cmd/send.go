package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/roboalchemist/tagents/pkg/session"
	"github.com/spf13/cobra"
)

var flagSendForce bool
var flagSendDryRun bool

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
	SuggestFor: []string{"msg", "message"},
	Args:       cobra.ExactArgs(2),
	RunE:       runSend,
}

func init() {
	sendCmd.Flags().BoolVarP(&flagSendForce, "force", "f", false, "Send even if agent is busy")
	sendCmd.Flags().BoolVar(&flagSendDryRun, "dry-run", false, "Print what would be sent without sending")
	rootCmd.AddCommand(sendCmd)
}

func runSend(cmd *cobra.Command, args []string) error {
	query := args[0]
	message := args[1]

	if flagSendDryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] Would send to %s: %s\n", query, message)
		return nil
	}

	s, err := findSession(query)
	if err != nil {
		return err
	}

	// Warn if agent is busy
	if s.Status == runtime.Busy && !flagSendForce {
		fmt.Fprintf(os.Stderr, "Warning: agent %q is busy. Use --force to send anyway.\n", s.Name)
		return fmt.Errorf("agent is busy (use --force to override)")
	}

	client, err := session.ClientFor(*s, 10*time.Second)
	if err != nil {
		return err
	}
	if err := client.SendKeys(s.Name, message); err != nil {
		return fmt.Errorf("sending to %q: %w", s.Name, err)
	}

	fmt.Printf("Sent to %s\n", s.Name)
	return nil
}
