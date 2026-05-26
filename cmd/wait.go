package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/roboalchemist/tagents/pkg/output"
	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/roboalchemist/tagents/pkg/session"
	"github.com/spf13/cobra"
)

var flagWaitTimeout time.Duration

var waitCmd = &cobra.Command{
	Use:   "wait [agent ...] [timeout]",
	Short: "Wait until any of the specified agents is idle",
	Long: `Block until at least one agent from the set becomes idle.

Accepts one or more agent names (fuzzy-matched). With no agent names and a
machine scope flag, waits for any idle agent on that machine or fleet.

Timeout can be given as a --timeout flag or as the last positional argument
(e.g. "60s", "2m", "1h"). The positional form is kept for backward compat.
Default: 60s. Exit 0 when an idle agent is found, exit 1 on timeout.

Examples:
  tagents wait my-agent
  tagents wait my-agent 120s
  tagents wait worker-1 worker-2 worker-3
  tagents wait worker-1 worker-2 --timeout 3m
  tagents wait --machine gateway
  tagents wait --all-machines --timeout 5m`,
	Example: `  tagents wait my-agent
  tagents wait worker-1 worker-2 3m
  tagents wait --all-machines`,
	SuggestFor: []string{"watch", "block", "poll"},
	Args:       cobra.ArbitraryArgs,
	RunE:       runWait,
}

func init() {
	waitCmd.Flags().DurationVarP(&flagWaitTimeout, "timeout", "t", 60*time.Second, "Polling timeout (e.g. 60s, 2m, 1h)")
	rootCmd.AddCommand(waitCmd)
}

func runWait(cmd *cobra.Command, args []string) error {
	queries, timeout := parseWaitArgs(args, flagWaitTimeout)

	machine, allMachines := GetMachineScope()
	fleetMode := len(queries) == 0

	if fleetMode && !allMachines && machine == "" {
		return fmt.Errorf("no agents specified: provide agent names or use --machine/--all-machines")
	}

	opts := GetOutputOptions()
	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	if fleetMode {
		scope := machine
		if allMachines {
			scope = "all machines"
		}
		fmt.Fprintf(os.Stderr, "Waiting for any idle agent on %s (timeout: %s)...\n", scope, timeout)
	} else {
		fmt.Fprintf(os.Stderr, "Waiting for any of [%s] to be idle (timeout: %s)...\n",
			strings.Join(queries, ", "), timeout)
	}

	for time.Now().Before(deadline) {
		sessions, err := getSessions(machine, allMachines)
		if err != nil {
			return err
		}

		candidates := resolveWaitCandidates(sessions, queries, fleetMode)

		for _, s := range candidates {
			if s.Status == runtime.Idle {
				ref := waitAgentRef(s)
				fmt.Fprintf(os.Stderr, "Agent %s is idle.\n", ref)
				if opts.Mode == output.ModeJSON {
					return output.RenderJSON(s, opts)
				}
				fmt.Println(ref)
				return nil
			}
		}

		if len(candidates) == 0 && !fleetMode {
			fmt.Fprintf(os.Stderr, "  (no matching agents found yet, retrying...)\n")
		} else {
			for _, s := range candidates {
				fmt.Fprintf(os.Stderr, "  %s: %s...\n", waitAgentRef(s), s.Status)
			}
		}

		time.Sleep(pollInterval)
	}

	if fleetMode {
		return fmt.Errorf("timeout: no idle agent found after %s", timeout)
	}
	return fmt.Errorf("timeout: none of [%s] became idle after %s", strings.Join(queries, ", "), timeout)
}

// parseWaitArgs splits positional args into agent queries and timeout.
// If the last arg parses as time.Duration it is the timeout (backward compat
// with the old `wait <agent> [timeout]` signature). Otherwise flagTimeout is used.
func parseWaitArgs(args []string, flagTimeout time.Duration) (queries []string, timeout time.Duration) {
	timeout = flagTimeout
	if len(args) > 0 {
		if d, err := time.ParseDuration(args[len(args)-1]); err == nil {
			timeout = d
			args = args[:len(args)-1]
		}
	}
	return args, timeout
}

// resolveWaitCandidates returns the sessions to monitor.
// In fleet mode every discovered session is a candidate.
// Otherwise each query is fuzzy-matched; unmatched queries are silently skipped
// (the session may not exist yet and will be retried next poll).
func resolveWaitCandidates(sessions []session.AgentSession, queries []string, fleetMode bool) []session.AgentSession {
	if fleetMode {
		return sessions
	}
	seen := make(map[string]bool)
	var result []session.AgentSession
	for _, q := range queries {
		s, err := session.FuzzyMatch(sessions, q)
		if err != nil {
			continue
		}
		key := s.Machine + ":" + s.Name
		if !seen[key] {
			seen[key] = true
			result = append(result, *s)
		}
	}
	return result
}

func waitAgentRef(s session.AgentSession) string {
	if s.Machine != "" {
		return s.Machine + ":" + s.Name
	}
	return s.Name
}
