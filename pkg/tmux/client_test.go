package tmux

import (
	"fmt"
	"strings"
	"testing"
)

// MockExecutor records calls and returns preset responses.
type MockExecutor struct {
	responses map[string]string
	errors    map[string]error
}

func newMock() *MockExecutor {
	return &MockExecutor{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *MockExecutor) set(args string, out string, err error) {
	m.responses[args] = out
	if err != nil {
		m.errors[args] = err
	}
}

func (m *MockExecutor) Run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if out, ok := m.responses[key]; ok {
		return out, nil
	}
	return "", fmt.Errorf("unexpected tmux call: %s", key)
}

func TestListSessions(t *testing.T) {
	mock := newMock()
	mock.set("list-sessions -F #{session_name}:#{session_windows}:#{session_attached}:#{session_created}",
		"main-agent:2:1:1716000000\nbg-worker:1:0:1716001000", nil)
	c := NewClient(mock)
	sessions, err := c.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Name != "main-agent" {
		t.Errorf("expected main-agent, got %s", sessions[0].Name)
	}
	if sessions[0].Windows != 2 {
		t.Errorf("expected 2 windows, got %d", sessions[0].Windows)
	}
	if !sessions[0].Attached {
		t.Error("expected attached=true")
	}
	if sessions[1].Attached {
		t.Error("expected attached=false for bg-worker")
	}
}

func TestListSessions_NoServer(t *testing.T) {
	mock := newMock()
	mock.set("list-sessions -F #{session_name}:#{session_windows}:#{session_attached}:#{session_created}",
		"", fmt.Errorf("tmux list-sessions: exit status 1: no server running on /tmp/tmux-1000/default"))
	c := NewClient(mock)
	sessions, err := c.ListSessions()
	if err != nil {
		t.Fatalf("expected nil error for no-server, got: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestListSessions_Empty(t *testing.T) {
	mock := newMock()
	mock.set("list-sessions -F #{session_name}:#{session_windows}:#{session_attached}:#{session_created}",
		"", nil)
	c := NewClient(mock)
	sessions, err := c.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 { //nolint:staticcheck
		t.Errorf("expected empty, got %v", sessions)
	}
}

func TestListSessions_MalformedLine(t *testing.T) {
	mock := newMock()
	mock.set("list-sessions -F #{session_name}:#{session_windows}:#{session_attached}:#{session_created}",
		"good-session:1:0:1716000000\nbad-line\ngood-session2:1:0:1716000001", nil)
	c := NewClient(mock)
	sessions, err := c.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// malformed line should be skipped, only 2 valid sessions returned
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions (bad line skipped), got %d", len(sessions))
	}
}

func TestListSessions_UnexpectedError(t *testing.T) {
	mock := newMock()
	mock.set("list-sessions -F #{session_name}:#{session_windows}:#{session_attached}:#{session_created}",
		"", fmt.Errorf("some unexpected fatal error"))
	c := NewClient(mock)
	_, err := c.ListSessions()
	if err == nil {
		t.Error("expected error for unexpected failure")
	}
}

func TestCapturePane(t *testing.T) {
	mock := newMock()
	mock.set("capture-pane -p -t my-agent -S -50", "line1\nline2\n❯ ", nil)
	c := NewClient(mock)
	out, err := c.CapturePane("my-agent", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "line1") {
		t.Errorf("expected line1 in output, got: %s", out)
	}
}

func TestCapturePane_Error(t *testing.T) {
	mock := newMock()
	mock.set("capture-pane -p -t missing -S -50", "", fmt.Errorf("exit status 1"))
	c := NewClient(mock)
	_, err := c.CapturePane("missing", 50)
	if err == nil {
		t.Error("expected error for missing session")
	}
}

func TestSendKeys(t *testing.T) {
	mock := newMock()
	mock.set("send-keys -t my-agent hello Enter", "", nil)
	c := NewClient(mock)
	if err := c.SendKeys("my-agent", "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendKeys_Error(t *testing.T) {
	mock := newMock()
	mock.set("send-keys -t gone hello Enter", "", fmt.Errorf("exit status 1"))
	c := NewClient(mock)
	if err := c.SendKeys("gone", "hello"); err == nil {
		t.Error("expected error for missing session")
	}
}

func TestGetPaneCWD(t *testing.T) {
	mock := newMock()
	mock.set("display-message -p -t my-agent #{pane_current_path}", "/home/user/project", nil)
	c := NewClient(mock)
	cwd, err := c.GetPaneCWD("my-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cwd != "/home/user/project" {
		t.Errorf("expected /home/user/project, got %s", cwd)
	}
}

func TestGetPaneCWD_Error(t *testing.T) {
	mock := newMock()
	mock.set("display-message -p -t gone #{pane_current_path}", "", fmt.Errorf("exit status 1"))
	c := NewClient(mock)
	_, err := c.GetPaneCWD("gone")
	if err == nil {
		t.Error("expected error for missing session")
	}
}

func TestSessionExists(t *testing.T) {
	mock := newMock()
	mock.set("has-session -t existing", "", nil)
	mock.set("has-session -t missing", "", fmt.Errorf("exit status 1"))
	c := NewClient(mock)
	if !c.SessionExists("existing") {
		t.Error("expected existing session to exist")
	}
	if c.SessionExists("missing") {
		t.Error("expected missing session to not exist")
	}
}

func TestIsPaneRunning(t *testing.T) {
	mock := newMock()
	mock.set("display-message -p -t busy-agent #{pane_current_command}", "node", nil)
	mock.set("display-message -p -t idle-agent #{pane_current_command}", "zsh", nil)
	c := NewClient(mock)
	running, err := c.IsPaneRunning("busy-agent")
	if err != nil {
		t.Fatal(err)
	}
	if !running {
		t.Error("expected node process to be running")
	}
	idle, err := c.IsPaneRunning("idle-agent")
	if err != nil {
		t.Fatal(err)
	}
	if idle {
		t.Error("expected zsh shell to be idle")
	}
}

func TestIsPaneRunning_AllShells(t *testing.T) {
	shells := []string{"bash", "sh", "fish", "dash"}
	for _, shell := range shells {
		mock := newMock()
		mock.set("display-message -p -t test-session #{pane_current_command}", shell, nil)
		c := NewClient(mock)
		running, err := c.IsPaneRunning("test-session")
		if err != nil {
			t.Fatalf("unexpected error for shell %s: %v", shell, err)
		}
		if running {
			t.Errorf("expected shell %s to be idle (not running)", shell)
		}
	}
}

func TestIsPaneRunning_Error(t *testing.T) {
	mock := newMock()
	mock.set("display-message -p -t gone #{pane_current_command}", "", fmt.Errorf("exit status 1"))
	c := NewClient(mock)
	_, err := c.IsPaneRunning("gone")
	if err == nil {
		t.Error("expected error for missing session")
	}
}

func TestNewLocalClient(t *testing.T) {
	c := NewLocalClient()
	if c == nil { //nolint:staticcheck
		t.Error("expected non-nil client")
	}
	if c.executor == nil { //nolint:staticcheck
		t.Error("expected non-nil executor")
	}
	// Verify it's an OSExecutor
	if _, ok := c.executor.(*OSExecutor); !ok { //nolint:staticcheck
		t.Error("expected OSExecutor")
	}
}
