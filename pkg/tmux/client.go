package tmux

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Client wraps tmux CLI commands.
type Client struct {
	executor Executor
}

// NewClient creates a Client with the given executor (use &OSExecutor{} for production).
func NewClient(exec Executor) *Client {
	return &Client{executor: exec}
}

// NewLocalClient creates a Client using the real tmux CLI.
func NewLocalClient() *Client {
	return &Client{executor: &OSExecutor{}}
}

// ListSessions returns all tmux sessions.
// Format string: #{session_name}:#{session_windows}:#{session_attached}:#{session_created}
func (c *Client) ListSessions() ([]Session, error) {
	out, err := c.executor.Run("list-sessions", "-F",
		"#{session_name}:#{session_windows}:#{session_attached}:#{session_created}")
	if err != nil {
		// tmux exits non-zero when no sessions — normalize to empty list
		if strings.Contains(err.Error(), "no server running") ||
			strings.Contains(err.Error(), "no sessions") ||
			strings.Contains(err.Error(), "exit status 1") {
			return nil, nil
		}
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var sessions []Session
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		s, err := parseSessionLine(line)
		if err != nil {
			continue // skip malformed lines
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func parseSessionLine(line string) (Session, error) {
	parts := strings.Split(line, ":")
	if len(parts) < 4 {
		return Session{}, fmt.Errorf("unexpected format: %q", line)
	}
	name := parts[0]
	windows, _ := strconv.Atoi(parts[1])
	attached := parts[2] == "1"
	createdUnix, _ := strconv.ParseInt(parts[3], 10, 64)
	created := time.Unix(createdUnix, 0)
	return Session{Name: name, Windows: windows, Attached: attached, Created: created}, nil
}

// CapturePane returns the last N lines from the named session's active pane.
func (c *Client) CapturePane(sessionName string, lines int) (string, error) {
	target := sessionName
	startLine := fmt.Sprintf("-%d", lines)
	out, err := c.executor.Run("capture-pane", "-p", "-t", target, "-S", startLine)
	if err != nil {
		return "", fmt.Errorf("capture-pane %s: %w", sessionName, err)
	}
	return out, nil
}

// SendKeys sends a message to a session's active pane as if typed + Enter.
func (c *Client) SendKeys(sessionName, message string) error {
	_, err := c.executor.Run("send-keys", "-t", sessionName, message, "Enter")
	return err
}

// GetPaneCWD returns the current working directory of the session's active pane.
func (c *Client) GetPaneCWD(sessionName string) (string, error) {
	out, err := c.executor.Run("display-message", "-p", "-t", sessionName, "#{pane_current_path}")
	if err != nil {
		return "", fmt.Errorf("display-message %s: %w", sessionName, err)
	}
	return strings.TrimSpace(out), nil
}

// GetPaneProcess returns the foreground command name and terminal title of the
// session's active pane. These are process-level signals: unlike pane content,
// they do not depend on which lines happen to be visible or on redraw timing.
// The title is application-controlled and may be empty.
func (c *Client) GetPaneProcess(sessionName string) (command, title string, err error) {
	out, err := c.executor.Run("display-message", "-p", "-t", sessionName,
		"#{pane_current_command}\t#{pane_title}")
	if err != nil {
		return "", "", fmt.Errorf("display-message %s: %w", sessionName, err)
	}
	parts := strings.SplitN(out, "\t", 2)
	command = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		title = strings.TrimSpace(parts[1])
	}
	return command, title, nil
}

// SessionExists returns true if a session with the given name exists.
func (c *Client) SessionExists(name string) bool {
	_, err := c.executor.Run("has-session", "-t", name)
	return err == nil
}

// NewSession creates a detached tmux session rooted at cwd. When cwd is empty
// tmux uses the server's default working directory.
func (c *Client) NewSession(name, cwd string) error {
	args := []string{"new-session", "-d", "-s", name}
	if cwd != "" {
		args = append(args, "-c", cwd)
	}
	if _, err := c.executor.Run(args...); err != nil {
		return fmt.Errorf("new-session %s: %w", name, err)
	}
	return nil
}

// KillSession terminates a tmux session.
func (c *Client) KillSession(name string) error {
	if _, err := c.executor.Run("kill-session", "-t", name); err != nil {
		return fmt.Errorf("kill-session %s: %w", name, err)
	}
	return nil
}

// IsPaneRunning returns true if the pane has a command currently running (not at a shell prompt).
// This checks the pane_current_command — if it's the shell itself, the pane is idle.
func (c *Client) IsPaneRunning(sessionName string) (bool, error) {
	out, err := c.executor.Run("display-message", "-p", "-t", sessionName, "#{pane_current_command}")
	if err != nil {
		return false, err
	}
	cmd := strings.TrimSpace(out)
	// Common shell names — if the active process is a shell, pane is idle at prompt
	shells := map[string]bool{"bash": true, "zsh": true, "sh": true, "fish": true, "dash": true}
	return !shells[cmd], nil
}
