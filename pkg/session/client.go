package session

import (
	"fmt"
	"time"

	"github.com/roboalchemist/tagents/pkg/ssh"
	"github.com/roboalchemist/tagents/pkg/tmux"
)

// ClientFor returns a tmux client for the machine the session lives on:
// a local client when s.Machine is empty, otherwise a remote client that
// runs tmux over SSH using the host found in the SSH config.
func ClientFor(s AgentSession, timeout time.Duration) (*tmux.Client, error) {
	if s.Machine == "" {
		return tmux.NewLocalClient(), nil
	}

	hosts, err := ssh.ParseSSHConfig("")
	if err != nil {
		return nil, fmt.Errorf("reading SSH config: %w", err)
	}
	for _, h := range hosts {
		if h.Name == s.Machine {
			return tmux.NewClient(&remoteExecutor{host: h, timeout: timeout}), nil
		}
	}
	return nil, fmt.Errorf("machine %q not found in SSH config", s.Machine)
}
