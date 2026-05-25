package session

import (
	"testing"

	"github.com/roboalchemist/tagents/pkg/runtime"
)

func makeSession(machine, name string) AgentSession {
	return AgentSession{
		Machine: machine,
		Name:    name,
		Runtime: runtime.Claude,
		Status:  runtime.Idle,
	}
}

func TestFuzzyMatch_ExactName(t *testing.T) {
	sessions := []AgentSession{
		makeSession("", "my-agent"),
		makeSession("", "other-agent"),
	}
	got, err := FuzzyMatch(sessions, "my-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "my-agent" {
		t.Errorf("expected my-agent, got %s", got.Name)
	}
}

func TestFuzzyMatch_Substring(t *testing.T) {
	sessions := []AgentSession{
		makeSession("", "oh-my-agent-1"),
	}
	// "agent-1" should match "oh-my-agent-1" by substring
	got, err := FuzzyMatch(sessions, "agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "oh-my-agent-1" {
		t.Errorf("expected oh-my-agent-1, got %s", got.Name)
	}
}

func TestFuzzyMatch_MachinePin(t *testing.T) {
	sessions := []AgentSession{
		makeSession("", "worker"),        // local
		makeSession("gateway", "worker"), // remote
	}
	got, err := FuzzyMatch(sessions, "gateway:worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Machine != "gateway" {
		t.Errorf("expected gateway machine, got %s", got.Machine)
	}
}

func TestFuzzyMatch_LocalPin(t *testing.T) {
	sessions := []AgentSession{
		makeSession("", "worker"),
		makeSession("gateway", "worker"),
	}
	got, err := FuzzyMatch(sessions, "local:worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Machine != "" {
		t.Errorf("expected local (empty machine), got %s", got.Machine)
	}
}

func TestFuzzyMatch_Ambiguous(t *testing.T) {
	sessions := []AgentSession{
		makeSession("", "worker-1"),
		makeSession("", "worker-2"),
	}
	// "worker" matches both
	_, err := FuzzyMatch(sessions, "worker")
	if err == nil {
		t.Error("expected error for ambiguous match")
	}
}

func TestFuzzyMatch_NotFound(t *testing.T) {
	sessions := []AgentSession{
		makeSession("", "worker-1"),
	}
	_, err := FuzzyMatch(sessions, "nonexistent")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestFuzzyMatch_Empty(t *testing.T) {
	_, err := FuzzyMatch(nil, "anything")
	if err == nil {
		t.Error("expected error for empty sessions")
	}
}

func TestFuzzyMatch_AmbiguousExactAcrossMachines(t *testing.T) {
	// Same name on two different machines — exact match is ambiguous without machine pin
	sessions := []AgentSession{
		makeSession("", "worker"),
		makeSession("gateway", "worker"),
	}
	_, err := FuzzyMatch(sessions, "worker")
	if err == nil {
		t.Error("expected error for ambiguous exact match across machines")
	}
}

func TestFormatRef(t *testing.T) {
	local := makeSession("", "my-agent")
	remote := makeSession("gateway", "my-agent")
	if got := formatRef(local); got != "my-agent" {
		t.Errorf("expected my-agent, got %s", got)
	}
	if got := formatRef(remote); got != "gateway:my-agent" {
		t.Errorf("expected gateway:my-agent, got %s", got)
	}
}

func TestExactMatch_NotFound(t *testing.T) {
	sessions := []AgentSession{makeSession("gateway", "worker")}
	_, err := exactMatch(sessions, "mini", "worker")
	if err == nil {
		t.Error("expected error for machine mismatch")
	}
}
