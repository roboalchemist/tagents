package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/roboalchemist/tagents/pkg/session"
)

func TestWaitReadyPaneIdle(t *testing.T) {
	s := session.AgentSession{Name: "agent", Status: runtime.Idle}
	ready, reason, _, _ := waitReady(s, 0, map[string]string{}, time.Now())
	if !ready {
		t.Fatal("pane-idle session should be ready")
	}
	if reason != "pane idle" {
		t.Fatalf("reason = %q, want pane idle", reason)
	}
}

func TestWaitReadyLogIdleDisabled(t *testing.T) {
	s := session.AgentSession{Name: "agent", Runtime: runtime.Claude, Status: runtime.Busy}
	ready, _, _, _ := waitReady(s, 0, map[string]string{}, time.Now())
	if ready {
		t.Fatal("busy session should not be ready when log-idle is disabled")
	}
}

func TestWaitReadyLogIdleThreshold(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	logPath := writeWaitLog(t, now.Add(-2*time.Minute))
	s := session.AgentSession{Name: "agent", Runtime: runtime.Claude, Status: runtime.Busy}

	ready, reason, idle, found := waitReady(s, 90*time.Second, map[string]string{"agent": logPath}, now)
	if !ready {
		t.Fatal("stale log should make busy session ready")
	}
	if !found {
		t.Fatal("expected logFound=true")
	}
	if idle != 2*time.Minute {
		t.Fatalf("idle = %v, want 2m", idle)
	}
	if reason == "" || reason == "pane idle" {
		t.Fatalf("reason = %q, want log-idle reason", reason)
	}
}

func TestWaitReadyLogIdleBelowThreshold(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	logPath := writeWaitLog(t, now.Add(-30*time.Second))
	s := session.AgentSession{Name: "agent", Runtime: runtime.Claude, Status: runtime.Busy}

	ready, _, idle, found := waitReady(s, 90*time.Second, map[string]string{"agent": logPath}, now)
	if ready {
		t.Fatal("fresh log should not make busy session ready")
	}
	if !found {
		t.Fatal("expected logFound=true")
	}
	if idle != 30*time.Second {
		t.Fatalf("idle = %v, want 30s", idle)
	}
}

func TestWaitReadyRemoteDoesNotReadLocalLog(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	logPath := writeWaitLog(t, now.Add(-2*time.Minute))
	s := session.AgentSession{Machine: "gateway", Name: "agent", Runtime: runtime.Claude, Status: runtime.Busy}

	ready, _, _, found := waitReady(s, 90*time.Second, map[string]string{"gateway:agent": logPath}, now)
	if ready {
		t.Fatal("remote busy session should not use local log-idle")
	}
	if found {
		t.Fatal("remote session should not report local log found")
	}
}

func writeWaitLog(t *testing.T, ts time.Time) string {
	t.Helper()
	path := t.TempDir() + "/session.jsonl"
	line := `{"timestamp":"` + ts.Format(time.RFC3339Nano) + `"}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
