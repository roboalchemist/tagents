package session

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/roboalchemist/tagents/pkg/ssh"
	"github.com/roboalchemist/tagents/pkg/tmux"
)

// DiscoverLocal discovers all agent sessions on the local machine.
func DiscoverLocal() ([]AgentSession, error) {
	client := tmux.NewLocalClient()
	return discoverOnClient("", client)
}

// DiscoverRemote discovers agent sessions on a remote host.
func DiscoverRemote(host ssh.Host, timeout time.Duration) ([]AgentSession, error) {
	executor := &remoteExecutor{host: host, timeout: timeout}
	client := tmux.NewClient(executor)
	return discoverOnClient(host.Name, client)
}

// DiscoverAll discovers sessions on local machine and all provided remote hosts in parallel.
// Unreachable hosts are silently skipped (warning printed to stderr by caller).
func DiscoverAll(hosts []ssh.Host, timeout time.Duration) ([]AgentSession, []error) {
	var mu sync.Mutex
	var all []AgentSession
	var errs []error

	// Local sessions
	local, err := DiscoverLocal()
	if err == nil {
		mu.Lock()
		all = append(all, local...)
		mu.Unlock()
	}

	// Remote sessions in parallel
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, h := range hosts {
		h := h
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions, err := DiscoverRemote(h, timeout)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			all = append(all, sessions...)
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}

	// Wait for any goroutines that may still be running after timeout,
	// then take a snapshot under the lock to avoid data races.
	wg.Wait()
	mu.Lock()
	allSnap := make([]AgentSession, len(all))
	copy(allSnap, all)
	errsSnap := make([]error, len(errs))
	copy(errsSnap, errs)
	mu.Unlock()
	return allSnap, errsSnap
}

// discoverOnClient gets all agent sessions from a tmux client (local or remote).
func discoverOnClient(machine string, client *tmux.Client) ([]AgentSession, error) {
	sessions, err := client.ListSessions()
	if err != nil {
		return nil, err
	}

	var result []AgentSession
	for _, s := range sessions {
		// Get pane content for runtime/status detection
		paneContent, _ := client.CapturePane(s.Name, 50)
		rt := runtime.DetectRuntime(s.Name, paneContent)
		status := runtime.DetectStatus(paneContent)
		cwd, _ := client.GetPaneCWD(s.Name)
		preview := lastMeaningfulLine(paneContent)

		result = append(result, AgentSession{
			Machine: machine,
			Name:    s.Name,
			Runtime: rt,
			Status:  status,
			CWD:     cwd,
			Preview: preview,
		})
	}
	return result, nil
}

// lastMeaningfulLine returns the last non-empty line from pane content, truncated to 80 chars.
func lastMeaningfulLine(content string) string {
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			if len(line) > 80 {
				return line[:77] + "..."
			}
			return line
		}
	}
	return ""
}

// remoteExecutor wraps ssh.RemoteExecutor to satisfy tmux.Executor interface.
type remoteExecutor struct {
	host    ssh.Host
	timeout time.Duration
}

func (r *remoteExecutor) Run(args ...string) (string, error) {
	// Build: tmux <args>
	command := "tmux " + strings.Join(shellEscape(args), " ")
	executor := &ssh.RemoteExecutor{}
	stdout, _, err := executor.RunTimeout(r.host, command, r.timeout)
	return strings.TrimRight(stdout, "\n"), err
}

// shellEscape quotes arguments that contain spaces or special characters.
func shellEscape(args []string) []string {
	result := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t\n'\"\\$`{}()|&;<>") {
			result[i] = "'" + strings.ReplaceAll(a, "'", "'\\''") + "'"
		} else {
			result[i] = a
		}
	}
	return result
}
