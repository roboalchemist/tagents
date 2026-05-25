package cmd

import (
	"fmt"
	"os"

	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/roboalchemist/tagents/pkg/tmux"
	"github.com/spf13/cobra"
)

var (
	flagBroadcastRuntime string
	flagBroadcastDryRun  bool
)

var broadcastCmd = &cobra.Command{
	Use:   "broadcast <message>",
	Short: "Send message to all idle agents",
	Long: `Send a message to every idle agent in scope.

Skips busy and dead agents. Use --runtime to target a specific agent type.
Use --dry-run to preview without sending.

Examples:
  tagents broadcast "please summarize your current status"
  tagents broadcast "stop" --runtime claude
  tagents broadcast "check in" --all-machines --dry-run`,
	Example: `  tagents broadcast "please continue"
  tagents broadcast "status check" --runtime claude --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runBroadcast,
}

func init() {
	broadcastCmd.Flags().StringVar(&flagBroadcastRuntime, "runtime", "", "Filter by runtime: claude or codex")
	broadcastCmd.Flags().BoolVar(&flagBroadcastDryRun, "dry-run", false, "Show what would be sent without sending")
	rootCmd.AddCommand(broadcastCmd)
}

func runBroadcast(cmd *cobra.Command, args []string) error {
	message := args[0]
	machine, allMachines := GetMachineScope()

	sessions, err := getSessions(machine, allMachines)
	if err != nil {
		return err
	}

	client := tmux.NewLocalClient()
	sent := 0
	skipped := 0

	for _, s := range sessions {
		// Filter by runtime if requested
		if flagBroadcastRuntime != "" && string(s.Runtime) != flagBroadcastRuntime {
			skipped++
			continue
		}
		// Only send to idle agents
		if s.Status != runtime.Idle {
			skipped++
			continue
		}

		if flagBroadcastDryRun {
			fmt.Printf("[dry-run] would send to %s\n", s.Name)
			sent++
			continue
		}

		if err := client.SendKeys(s.Name, message); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to send to %s: %v\n", s.Name, err)
			skipped++
			continue
		}
		fmt.Printf("Sent to %s\n", s.Name)
		sent++
	}

	fmt.Printf("\nSent to %d agents, skipped %d\n", sent, skipped)
	return nil
}
