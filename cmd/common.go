package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/roboalchemist/tagents/pkg/output"
	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/roboalchemist/tagents/pkg/session"
	"github.com/roboalchemist/tagents/pkg/ssh"
	"github.com/fatih/color"
)

// getSessions discovers agent sessions based on scope flags.
// If allMachines is true, discovers local + all SSH config hosts.
// If machine is set, discovers only that host.
// Otherwise discovers local only.
func getSessions(machine string, allMachines bool) ([]session.AgentSession, error) {
	if !allMachines && machine == "" {
		return session.DiscoverLocal()
	}

	sshHosts, err := ssh.ParseSSHConfig("")
	if err != nil {
		return nil, fmt.Errorf("reading SSH config: %w", err)
	}

	if machine != "" && !allMachines {
		var filtered []ssh.Host
		for _, h := range sshHosts {
			if h.Name == machine {
				filtered = append(filtered, h)
				break
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("machine %q not found in SSH config", machine)
		}
		sshHosts = filtered
	}

	sessions, errs := session.DiscoverAll(sshHosts, 10*time.Second)
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "warning: %v\n", e)
	}
	return sessions, nil
}

// withLogIdle annotates local sessions with transcript staleness. Remote sessions
// are left unset because their logs live on the remote host.
func withLogIdle(sessions []session.AgentSession) []session.AgentSession {
	now := time.Now()
	for i := range sessions {
		if sessions[i].Machine != "" {
			continue
		}
		idle, found := runtime.LogIdleDuration(sessions[i].Name, sessions[i].Runtime, now)
		sessions[i].LogIdle = idle
		sessions[i].LogIdleFound = found
	}
	return sessions
}

func formatLogIdle(s session.AgentSession) string {
	if !s.LogIdleFound {
		return "-"
	}
	return formatDuration(s.LogIdle)
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Second {
		return d.Truncate(time.Millisecond).String()
	}
	return d.Truncate(time.Second).String()
}

// findSession finds a single session by fuzzy name match using the current scope flags.
// When no scope flags are given, it tries local sessions first (fast path) and
// falls back to fleet-wide discovery so remote agents are reachable without
// --all-machines.
func findSession(query string) (*session.AgentSession, error) {
	machine, allMachines := GetMachineScope()
	if machine == "" && !allMachines {
		if local, err := session.DiscoverLocal(); err == nil {
			if s, err := session.FuzzyMatch(local, query); err == nil {
				return s, nil
			}
		}
		allMachines = true
	}
	sessions, err := getSessions(machine, allMachines)
	if err != nil {
		return nil, err
	}
	return session.FuzzyMatch(sessions, query)
}

// truncate shortens s to at most n bytes, appending "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// colorStatus returns the status string, optionally colorized.
func colorStatus(status string, opts output.Options) string {
	if !opts.ShouldUseColor() {
		return status
	}
	switch status {
	case "idle":
		return color.GreenString(status)
	case "busy":
		return color.YellowString(status)
	case "dead":
		return color.RedString(status)
	default:
		return status
	}
}
