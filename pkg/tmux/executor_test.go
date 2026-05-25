package tmux

import (
	"os/exec"
	"strings"
	"testing"
)

func TestOSExecutor_Run_Success(t *testing.T) {
	// Verify tmux is available; skip if not
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}
	e := &OSExecutor{}
	// tmux -V returns the version string and exits 0
	out, err := e.Run("-V")
	if err != nil {
		t.Fatalf("expected success from tmux -V, got: %v", err)
	}
	if !strings.HasPrefix(out, "tmux ") {
		t.Errorf("expected version string starting with 'tmux ', got: %q", out)
	}
}

func TestOSExecutor_Run_Error(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}
	e := &OSExecutor{}
	// Pass a command that will definitely fail
	_, err := e.Run("has-session", "-t", "nonexistent-session-that-cannot-exist-xyz123")
	if err == nil {
		t.Error("expected error for nonexistent session")
	}
	// Error should contain the tmux args
	if !strings.Contains(err.Error(), "has-session") {
		t.Errorf("expected error to mention 'has-session', got: %v", err)
	}
}

func TestOSExecutor_Run_ErrorNoStderr(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available in PATH")
	}
	e := &OSExecutor{}
	// has-session on a missing session exits 1 with output on stderr
	_, err := e.Run("has-session", "-t", "___no_such_session___")
	if err == nil {
		t.Error("expected error")
	}
	// Error wraps the exit error
	if !strings.Contains(err.Error(), "tmux has-session") {
		t.Errorf("expected 'tmux has-session' in error, got: %v", err)
	}
}
