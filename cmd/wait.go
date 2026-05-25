package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/roboalchemist/tagents/pkg/session"
	"github.com/roboalchemist/tagents/pkg/tmux"
	"github.com/spf13/cobra"
)

var waitCmd = &cobra.Command{
	Use:   "wait <agent> [timeout]",
	Short: "Wait until agent is idle",
	Long: `Block polling until the agent's status is idle.

Timeout format: "60s", "2m", "1h". Default: 60s.
Exit 0 when idle, exit 1 on timeout.

Examples:
  tagents wait my-agent
  tagents wait my-agent 120s
  tagents wait my-agent 5m`,
	Example: `  tagents wait my-agent
  tagents wait my-agent 120s`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runWait,
}

func init() { rootCmd.AddCommand(waitCmd) }

func runWait(cmd *cobra.Command, args []string) error {
	query := args[0]
	timeout := 60 * time.Second

	if len(args) == 2 {
		var err error
		timeout, err = time.ParseDuration(args[1])
		if err != nil {
			return fmt.Errorf("invalid timeout %q: must be a duration like '60s', '2m', '1h'", args[1])
		}
	}

	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	fmt.Fprintf(os.Stderr, "Waiting for %s to be idle (timeout: %s)...\n", query, timeout)

	for time.Now().Before(deadline) {
		// Re-discover each poll to get fresh status
		machine, allMachines := GetMachineScope()
		sessions, err := getSessions(machine, allMachines)
		if err != nil {
			return err
		}

		s, err := session.FuzzyMatch(sessions, query)
		if err != nil {
			return err
		}

		// Get fresh pane content for status check
		client := tmux.NewLocalClient()
		paneContent, _ := client.CapturePane(s.Name, 20)
		status := runtime.DetectStatus(paneContent)

		if status == runtime.Idle {
			fmt.Fprintf(os.Stderr, "Agent %s is idle.\n", s.Name)
			return nil
		}

		fmt.Fprintf(os.Stderr, "  %s: %s...\n", s.Name, status)
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout: %s is still not idle after %s", query, timeout)
}
