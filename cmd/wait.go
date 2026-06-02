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
var flagWaitLogIdle time.Duration

var waitCmd = &cobra.Command{
	Use:   "wait [agent ...] [timeout]",
	Short: "Wait until any of the specified agents is ready",
	Long: `Block until at least one agent from the set becomes ready.

Accepts one or more agent names (fuzzy-matched). With no agent names and a
machine scope flag, waits for any ready agent on that machine or fleet.

Timeout can be given as a --timeout flag or as the last positional argument
(e.g. "60s", "2m", "1h"). The positional form is kept for backward compat.
Default: 60s. Exit 0 when an agent is ready, exit 1 on timeout.

With --log-idle, a busy-looking agent is also considered ready when its
Claude/Codex transcript has not advanced for at least the given duration. This
catches empty-turn/context-wedge stalls that pane-based idle detection misses.

Examples:
  tagents wait my-agent
  tagents wait my-agent 120s
  tagents wait worker-1 worker-2 worker-3
  tagents wait worker-1 worker-2 --timeout 3m
  tagents wait my-agent --log-idle 90s --timeout 30m
  tagents wait --machine gateway
  tagents wait --all-machines --timeout 5m`,
	Example: `  tagents wait my-agent
  tagents wait worker-1 worker-2 3m
  tagents wait my-agent --log-idle 90s --timeout 30m
  tagents wait --all-machines`,
	SuggestFor: []string{"watch", "block", "poll"},
	Args:       cobra.ArbitraryArgs,
	RunE:       runWait,
}

func init() {
	waitCmd.Flags().DurationVarP(&flagWaitTimeout, "timeout", "t", 60*time.Second, "Polling timeout (e.g. 60s, 2m, 1h)")
	waitCmd.Flags().DurationVar(&flagWaitLogIdle, "log-idle", 0, "Treat an agent as ready when its transcript has been stale for this duration (0 disables)")
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
		fmt.Fprintf(os.Stderr, "Waiting for any ready agent on %s (timeout: %s%s)...\n", scope, timeout, logIdleSuffix(flagWaitLogIdle))
	} else {
		fmt.Fprintf(os.Stderr, "Waiting for any of [%s] to be ready (timeout: %s%s)...\n",
			strings.Join(queries, ", "), timeout, logIdleSuffix(flagWaitLogIdle))
	}

	logPaths := make(map[string]string)

	for time.Now().Before(deadline) {
		sessions, err := getSessions(machine, allMachines)
		if err != nil {
			return err
		}

		candidates := resolveWaitCandidates(sessions, queries, fleetMode)

		for _, s := range candidates {
			ready, reason, logIdle, logFound := waitReady(s, flagWaitLogIdle, logPaths, time.Now())
			if ready {
				ref := waitAgentRef(s)
				s.LogIdle = logIdle
				s.LogIdleFound = logFound
				s.ReadyReason = reason
				fmt.Fprintf(os.Stderr, "Agent %s is ready (%s).\n", ref, reason)
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
				detail := string(s.Status)
				if flagWaitLogIdle > 0 {
					_, _, logIdle, logFound := waitReady(s, flagWaitLogIdle, logPaths, time.Now())
					if logFound {
						detail = fmt.Sprintf("%s, log idle %s", detail, formatDuration(logIdle))
					}
				}
				fmt.Fprintf(os.Stderr, "  %s: %s...\n", waitAgentRef(s), detail)
			}
		}

		time.Sleep(pollInterval)
	}

	if fleetMode {
		return fmt.Errorf("timeout: no ready agent found after %s", timeout)
	}
	return fmt.Errorf("timeout: none of [%s] became ready after %s", strings.Join(queries, ", "), timeout)
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

func waitReady(s session.AgentSession, threshold time.Duration, logPaths map[string]string, now time.Time) (ready bool, reason string, logIdle time.Duration, logFound bool) {
	if s.Status == runtime.Idle {
		return true, "pane idle", s.LogIdle, s.LogIdleFound
	}
	if threshold <= 0 || s.Machine != "" {
		return false, "", s.LogIdle, s.LogIdleFound
	}
	ref := waitAgentRef(s)
	logPath, ok := logPaths[ref]
	if !ok {
		logPath = runtime.FindLogFile(s.Name, s.Runtime)
		if logPath != "" {
			logPaths[ref] = logPath
		}
	}
	logIdle, logFound = runtime.LogIdleDurationFromPath(logPath, now)
	if logFound && logIdle >= threshold {
		return true, fmt.Sprintf("log idle %s >= %s", formatDuration(logIdle), threshold), logIdle, true
	}
	return false, "", logIdle, logFound
}

func logIdleSuffix(threshold time.Duration) string {
	if threshold <= 0 {
		return ""
	}
	return fmt.Sprintf(", log-idle: %s", threshold)
}
