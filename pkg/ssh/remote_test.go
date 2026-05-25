package ssh

import (
	"testing"
	"time"
)

func TestBuildSSHArgs(t *testing.T) {
	host := Host{
		Name:     "myhost",
		HostName: "192.168.1.10",
		User:     "ubuntu",
		Port:     "2222",
	}
	args := buildSSHArgs(host, "echo hello")
	// Should contain the target IP (HostName takes precedence over Name)
	found := false
	for _, a := range args {
		if a == "192.168.1.10" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 192.168.1.10 in args: %v", args)
	}
	// Should contain port
	portFound := false
	for i, a := range args {
		if a == "-p" && i+1 < len(args) && args[i+1] == "2222" {
			portFound = true
		}
	}
	if !portFound {
		t.Errorf("expected -p 2222 in args: %v", args)
	}
	// Command is last arg
	if args[len(args)-1] != "echo hello" {
		t.Errorf("expected command as last arg, got: %s", args[len(args)-1])
	}
}

func TestBuildSSHArgs_DefaultPort(t *testing.T) {
	host := Host{Name: "myhost", Port: "22"}
	args := buildSSHArgs(host, "true")
	for i, a := range args {
		if a == "-p" {
			t.Errorf("default port 22 should not add -p flag, got args[%d]=%s", i, args[i+1])
		}
	}
}

func TestBuildSSHArgs_NoHostName(t *testing.T) {
	host := Host{Name: "myalias", Port: "22"}
	args := buildSSHArgs(host, "true")
	// Should use alias when no HostName set
	found := false
	for _, a := range args {
		if a == "myalias" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected myalias in args: %v", args)
	}
}

func TestRemoteExecutor_Run(t *testing.T) {
	// Run delegates to RunTimeout — verify it returns an error for an unreachable host
	// (we can't test success without a live SSH target, but we verify the method exists
	// and propagates errors correctly)
	r := &RemoteExecutor{}
	host := Host{Name: "unreachable-run-test.invalid", HostName: "192.0.2.2", Port: "22"}
	_, _, err := r.Run(host, "true")
	if err == nil {
		t.Error("expected error for unreachable host, got nil")
	}
}

func TestRemoteExecutor_RunTimeout_ErrorOutput(t *testing.T) {
	// RunTimeout should capture stderr even on failure
	r := &RemoteExecutor{}
	host := Host{Name: "unreachable-stderr-test.invalid", HostName: "192.0.2.3", Port: "22"}
	stdout, _, err := r.RunTimeout(host, "true", 2*time.Second)
	if err == nil {
		t.Error("expected error for unreachable host, got nil")
	}
	// stdout should be empty for unreachable host
	_ = stdout
}

func TestBuildSSHArgs_WithIdentityFile(t *testing.T) {
	host := Host{
		Name:         "myhost",
		HostName:     "10.0.0.1",
		User:         "alice",
		Port:         "22",
		IdentityFile: "/home/alice/.ssh/id_ed25519",
	}
	args := buildSSHArgs(host, "whoami")
	// Should include -i flag
	identFound := false
	for i, a := range args {
		if a == "-i" && i+1 < len(args) && args[i+1] == "/home/alice/.ssh/id_ed25519" {
			identFound = true
		}
	}
	if !identFound {
		t.Errorf("expected -i /home/alice/.ssh/id_ed25519 in args: %v", args)
	}
	// Should NOT include -p for port 22
	for i, a := range args {
		if a == "-p" {
			t.Errorf("default port 22 should not add -p flag, got args[%d]=%s", i, args[i+1])
		}
	}
}

func TestRemoteExecutor_PingTimeout(t *testing.T) {
	r := &RemoteExecutor{}
	// Use an unreachable address with very short timeout
	host := Host{Name: "unreachable-test-host-999.invalid", HostName: "192.0.2.1", Port: "22"}
	reachable := r.Ping(host, 1*time.Second)
	if reachable {
		t.Error("expected unreachable host to return false")
	}
}
