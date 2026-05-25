package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/roboalchemist/tagents/pkg/output"
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

// truncate shortens s to at most n runes, appending "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
