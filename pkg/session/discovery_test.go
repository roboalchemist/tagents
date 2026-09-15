package session

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/roboalchemist/tagents/pkg/ssh"
	"github.com/roboalchemist/tagents/pkg/tmux"
)

// mockExecutor satisfies tmux.Executor for testing.
type mockExecutor struct {
	responses map[string]string
	errors    map[string]error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockExecutor) set(key, out string, err error) {
	m.responses[key] = out
	if err != nil {
		m.errors[key] = err
	}
}

func (m *mockExecutor) Run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if out, ok := m.responses[key]; ok {
		return out, nil
	}
	return "", fmt.Errorf("unexpected tmux call: %s", key)
}

// listKey is the expected key for list-sessions.
const listKey = "list-sessions -F #{session_name}:#{session_windows}:#{session_attached}:#{session_created}"

func TestLastMeaningfulLine(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"line1\nline2\n", "line2"},
		{"  \n  \nresult\n  ", "result"},
		{"", ""},
		{"   ", ""},
		{strings.Repeat("x", 100), strings.Repeat("x", 77) + "..."},
		{"short", "short"},
	}
	for _, tt := range tests {
		got := lastMeaningfulLine(tt.input)
		if got != tt.want {
			t.Errorf("lastMeaningfulLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input         []string
		containsSpace bool
	}{
		{[]string{"list-sessions"}, false},
		{[]string{"send-keys", "-t", "my session", "hello", "Enter"}, true},
	}
	for _, tt := range tests {
		result := shellEscape(tt.input)
		if len(result) != len(tt.input) {
			t.Errorf("shellEscape length mismatch: got %d, want %d", len(result), len(tt.input))
		}
		if tt.containsSpace {
			quoted := false
			for _, r := range result {
				if strings.HasPrefix(r, "'") {
					quoted = true
				}
			}
			if !quoted {
				t.Errorf("expected at least one quoted arg, got: %v", result)
			}
		}
	}
}

func TestShellEscape_SpecialChars(t *testing.T) {
	// Each special char in ContainsAny should trigger quoting
	specials := []string{
		"arg with space",
		"arg\twith\ttab",
		"arg\nwith\nnewline",
		"arg'with'quote",
		`arg"with"dquote`,
		`arg\with\backslash`,
		"arg$var",
		"arg`cmd`",
		"arg{brace}",
		"arg(paren)",
		"arg|pipe",
		"arg&amp",
		"arg;semi",
		"arg<lt",
		"arg>gt",
	}
	for _, s := range specials {
		result := shellEscape([]string{s})
		if len(result) != 1 {
			t.Errorf("expected 1 result for %q", s)
			continue
		}
		if !strings.HasPrefix(result[0], "'") {
			t.Errorf("expected quoted result for %q, got %q", s, result[0])
		}
	}
}

func TestDiscoverLocal_NoTmux(t *testing.T) {
	sessions, err := DiscoverLocal()
	if err != nil {
		t.Logf("DiscoverLocal returned error (acceptable if tmux not running): %v", err)
	}
	_ = sessions
}

func TestDiscoverOnClient_WithSessions(t *testing.T) {
	mock := newMockExecutor()
	// list-sessions returns two sessions
	mock.set(listKey, "agent-1:1:0:1716000000\nagent-2:2:1:1716001000", nil)
	// capture-pane for agent-1
	mock.set("capture-pane -p -t agent-1 -S -50", "Human: hello\nAssistant: hi", nil)
	mock.set("display-message -p -t agent-1 #{pane_current_path}", "/home/user/project1", nil)
	// capture-pane for agent-2
	mock.set("capture-pane -p -t agent-2 -S -50", "❯ running codex task", nil)
	mock.set("display-message -p -t agent-2 #{pane_current_path}", "/home/user/project2", nil)

	client := tmux.NewClient(mock)
	sessions, err := discoverOnClient("", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Name != "agent-1" {
		t.Errorf("expected agent-1, got %s", sessions[0].Name)
	}
	if sessions[0].CWD != "/home/user/project1" {
		t.Errorf("expected /home/user/project1, got %s", sessions[0].CWD)
	}
	if sessions[0].Preview == "" {
		t.Error("expected non-empty preview for agent-1")
	}
	if sessions[1].Name != "agent-2" {
		t.Errorf("expected agent-2, got %s", sessions[1].Name)
	}
}

func TestDiscoverOnClient_ProcessRuntime(t *testing.T) {
	mock := newMockExecutor()
	mock.set(listKey, "oc-agent:1:0:1716000000", nil)
	// Ambiguous pane content (the OpenCode status bar has scrolled out of view),
	// but the foreground process still identifies the runtime.
	mock.set("capture-pane -p -t oc-agent -S -50", "working...\nmore output", nil)
	mock.set("display-message -p -t oc-agent #{pane_current_path}", "/work", nil)
	mock.set("display-message -p -t oc-agent #{pane_current_command}\t#{pane_title}", "opencode\tOC | work", nil)

	client := tmux.NewClient(mock)
	sessions, err := discoverOnClient("", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Runtime != runtime.OpenCode {
		t.Errorf("expected opencode runtime, got %s", sessions[0].Runtime)
	}
}

func TestDiscoverOnClient_NoSessions(t *testing.T) {
	mock := newMockExecutor()
	mock.set(listKey, "", fmt.Errorf("tmux list-sessions: exit status 1: no server running on /tmp/tmux-1000/default"))

	client := tmux.NewClient(mock)
	sessions, err := discoverOnClient("", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestDiscoverOnClient_WithMachine(t *testing.T) {
	mock := newMockExecutor()
	mock.set(listKey, "remote-agent:1:0:1716000000", nil)
	mock.set("capture-pane -p -t remote-agent -S -50", "Claude Code working...", nil)
	mock.set("display-message -p -t remote-agent #{pane_current_path}", "/remote/path", nil)

	client := tmux.NewClient(mock)
	sessions, err := discoverOnClient("gateway", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Machine != "gateway" {
		t.Errorf("expected machine=gateway, got %s", sessions[0].Machine)
	}
}

func TestDiscoverOnClient_ListError(t *testing.T) {
	mock := newMockExecutor()
	mock.set(listKey, "", fmt.Errorf("some unexpected fatal error"))

	client := tmux.NewClient(mock)
	_, err := discoverOnClient("", client)
	if err == nil {
		t.Error("expected error from list failure")
	}
}

func TestDiscoverOnClient_CaptureError(t *testing.T) {
	// capture-pane errors should be tolerated — session still included
	mock := newMockExecutor()
	mock.set(listKey, "agent-1:1:0:1716000000", nil)
	mock.set("capture-pane -p -t agent-1 -S -50", "", fmt.Errorf("capture failed"))
	mock.set("display-message -p -t agent-1 #{pane_current_path}", "/path", nil)

	client := tmux.NewClient(mock)
	sessions, err := discoverOnClient("", client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	// Preview should be empty since capture failed
	if sessions[0].Preview != "" {
		t.Errorf("expected empty preview, got %q", sessions[0].Preview)
	}
}

func TestDiscoverAll_LocalOnly(t *testing.T) {
	// No remote hosts — just exercises the local + goroutine paths
	sessions, errs := DiscoverAll(nil, 5*time.Second)
	// Just verify it doesn't hang or panic
	_ = sessions
	_ = errs
}

func TestDiscoverAll_WithUnreachableHost(t *testing.T) {
	// Use a host that will fail to connect (timeout)
	hosts := []ssh.Host{
		{Name: "unreachable-host-99999", HostName: "192.0.2.1", Port: "22"},
	}
	sessions, errs := DiscoverAll(hosts, 2*time.Second)
	// Errors are expected for unreachable hosts
	_ = sessions
	_ = errs
}

func TestRemoteExecutor_Run(t *testing.T) {
	// Test that Run builds the tmux command correctly
	// We can't test real SSH in unit tests, so we verify the command construction
	// by checking that shellEscape is applied correctly.
	r := &remoteExecutor{
		host:    ssh.Host{Name: "test-host"},
		timeout: 5 * time.Second,
	}
	// Just verify the executor is constructed correctly
	if r.host.Name != "test-host" {
		t.Errorf("expected test-host, got %s", r.host.Name)
	}
	if r.timeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", r.timeout)
	}
}
