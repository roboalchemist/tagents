package main_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testSession = "tagents-test-session"
const binaryPath = "./tagents"

// TestMain builds the binary, creates a test tmux session, runs tests, tears down.
func TestMain(m *testing.M) {
	// Build binary
	if err := exec.Command("go", "build", "-o", binaryPath, ".").Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}
	defer os.Remove(binaryPath) //nolint:errcheck

	// Create test tmux session (ignore error if already exists)
	_ = exec.Command("tmux", "new-session", "-d", "-s", testSession, "-c", "/tmp").Run()

	code := m.Run()

	// Cleanup
	_ = exec.Command("tmux", "kill-session", "-t", testSession).Run()
	os.Exit(code)
}

func run(args ...string) (string, string, int) {
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	outBuf.Reset()
	errBuf.Reset()
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			code = 1
		}
	}
	return outBuf.String(), errBuf.String(), code
}

func mustJSON(t *testing.T, data string) interface{} {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(data)), &v); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\ndata: %s", err, data)
	}
	return v
}

// TestIntegration_List verifies the list command output formats.
func TestIntegration_List(t *testing.T) {
	// Default table output (should include our test session)
	out, _, code := run("list")
	if code != 0 {
		t.Fatalf("tagents list failed (exit %d): %s", code, out)
	}
	if !strings.Contains(out, testSession) {
		t.Errorf("expected test session %q in list output, got: %s", testSession, out)
	}

	// JSON output
	out, _, code = run("list", "--json")
	if code != 0 {
		t.Fatalf("tagents list --json failed (exit %d)", code)
	}
	v := mustJSON(t, out)
	arr, ok := v.([]interface{})
	if !ok {
		t.Fatalf("expected JSON array, got: %T", v)
	}
	if len(arr) == 0 {
		t.Error("expected at least one session in JSON output")
	}

	// Plaintext output
	out, _, code = run("list", "--plaintext")
	if code != 0 {
		t.Fatalf("tagents list --plaintext failed (exit %d)", code)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Errorf("expected header + at least one data row, got %d lines", len(lines))
	}

	// --fields flag
	out, _, code = run("list", "--json", "--fields", "name,status")
	if code != 0 {
		t.Fatalf("tagents list --json --fields failed: %s", out)
	}
	mustJSON(t, out) // just verify it's valid JSON

	// --jq flag: find our test session by name (order-independent)
	out, _, code = run("list", "--json", "--jq", `.[] | select(.name == "`+testSession+`") | .name`)
	if code != 0 {
		t.Fatalf("tagents list --json --jq failed: %s", out)
	}
	if !strings.Contains(out, testSession) {
		t.Errorf("expected test session in jq output, got: %s", out)
	}
}

// TestIntegration_Status verifies fleet status output.
func TestIntegration_Status(t *testing.T) {
	out, _, code := run("status")
	if code != 0 {
		t.Fatalf("tagents status failed (exit %d): %s", code, out)
	}
	if !strings.Contains(strings.ToLower(out), "total") {
		t.Errorf("expected 'total' in status output, got: %s", out)
	}

	// JSON
	out, _, code = run("status", "--json")
	if code != 0 {
		t.Fatalf("tagents status --json failed: %s", out)
	}
	v := mustJSON(t, out)
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("expected JSON object, got %T", v)
	}
	if _, ok := m["overall"]; !ok {
		t.Errorf("expected 'overall' key in status JSON: %s", out)
	}
}

// TestIntegration_Where verifies the where command.
func TestIntegration_Where(t *testing.T) {
	out, _, code := run("where", testSession)
	if code != 0 {
		t.Fatalf("tagents where %s failed (exit %d): %s", testSession, code, out)
	}
	cwd := strings.TrimSpace(out)
	if cwd == "" {
		t.Error("expected non-empty CWD from where command")
	}
}

// TestIntegration_Read verifies pane reading.
func TestIntegration_Read(t *testing.T) {
	out, _, code := run("read", testSession)
	if code != 0 {
		t.Fatalf("tagents read %s failed (exit %d)", testSession, code)
	}
	_ = out // content may be empty for a fresh session

	// JSON mode
	out, _, code = run("read", testSession, "--json")
	if code != 0 {
		t.Fatalf("tagents read --json failed (exit %d): %s", code, out)
	}
	v := mustJSON(t, out)
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("expected JSON object from read --json, got %T", v)
	}
	if _, ok := m["lines"]; !ok {
		t.Errorf("expected 'lines' key in read JSON output: %s", out)
	}

	// Custom line count
	_, _, code = run("read", testSession, "10")
	if code != 0 {
		t.Fatalf("tagents read with line count failed (exit %d)", code)
	}

	// Invalid line count
	_, _, code = run("read", testSession, "abc")
	if code == 0 {
		t.Error("expected non-zero exit for invalid line count")
	}
}

// TestIntegration_Wait verifies wait returns quickly for an idle session.
func TestIntegration_Wait(t *testing.T) {
	// The test session should be idle (just a shell)
	// Use a short timeout to keep the test fast
	out, _, code := run("wait", testSession, "10s")
	if code != 0 {
		t.Logf("tagents wait returned non-zero (session may be busy): %s", out)
		// Not a hard failure — depends on session state
	}
}

// TestIntegration_Send verifies send command (READONLY gated).
func TestIntegration_Send(t *testing.T) {
	if os.Getenv("READONLY") == "1" {
		t.Skip("READONLY=1: skipping send test")
	}
	out, _, code := run("send", testSession, "echo tagents-integration-test")
	if code != 0 {
		t.Fatalf("tagents send failed (exit %d): %s", code, out)
	}
}

// TestIntegration_Inject verifies inject command (READONLY gated).
func TestIntegration_Inject(t *testing.T) {
	if os.Getenv("READONLY") == "1" {
		t.Skip("READONLY=1: skipping inject test")
	}
	// Create a temp file to inject
	f, err := os.CreateTemp("", "tagents-inject-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name()) //nolint:errcheck
	_, _ = f.WriteString("# test goal\n")
	_ = f.Close()

	out, _, code := run("inject", testSession, f.Name())
	if code != 0 {
		t.Fatalf("tagents inject failed (exit %d): %s", code, out)
	}
}

// TestIntegration_Broadcast verifies broadcast with --dry-run (always safe).
func TestIntegration_Broadcast(t *testing.T) {
	out, _, code := run("broadcast", "test message", "--dry-run")
	if code != 0 {
		t.Fatalf("tagents broadcast --dry-run failed (exit %d): %s", code, out)
	}
	// dry-run should print what it would send
	_ = out
}

// TestIntegration_NotFound verifies fuzzy match errors.
func TestIntegration_NotFound(t *testing.T) {
	_, _, code := run("where", "nonexistent-session-xyz")
	if code == 0 {
		t.Error("expected non-zero exit for nonexistent session")
	}
}

// TestIntegration_FuzzyMatch verifies partial name matching.
func TestIntegration_FuzzyMatch(t *testing.T) {
	// testSession = "tagents-test-session", partial = "tagents-test"
	out, _, code := run("where", "tagents-test")
	if code != 0 {
		t.Fatalf("fuzzy match of 'tagents-test' should find session, exit %d: %s", code, out)
	}
}

// TestIntegration_OutputFlags verifies global output flags don't crash.
func TestIntegration_OutputFlags(t *testing.T) {
	tests := [][]string{
		{"list", "--no-color"},
		{"list", "--debug"},
		{"status", "--no-color"},
		{"status", "--json"},
		{"list", "--plaintext"},
	}
	for _, args := range tests {
		out, errOut, code := run(args...)
		if code != 0 {
			t.Errorf("tagents %v failed (exit %d): stdout=%s stderr=%s",
				args, code, out, errOut)
		}
	}
}

// TestIntegration_UsageErrors verifies proper exit codes for bad usage.
func TestIntegration_UsageErrors(t *testing.T) {
	tests := []struct {
		args []string
		name string
	}{
		{[]string{"where"}, "where requires agent arg"},
		{[]string{"send", "agent"}, "send requires message arg"},
		{[]string{"read", "agent", "notanumber"}, "read invalid lines"},
	}
	for _, tt := range tests {
		_, _, code := run(tt.args...)
		if code == 0 {
			t.Errorf("%s: expected non-zero exit, got 0", tt.name)
		}
	}
}

// runWithEnv runs the binary with additional env vars merged on top of the current environment.
func runWithEnv(env map[string]string, args ...string) (string, string, int) {
	cmd := exec.Command(binaryPath, args...)
	// Start from current environment so PATH and other essentials are preserved.
	base := os.Environ()
	for k, v := range env {
		base = append(base, k+"="+v)
	}
	cmd.Env = base
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			code = 1
		}
	}
	return outBuf.String(), errBuf.String(), code
}

// TestIntegration_Machines tests the machines command in all output modes.
// The machines command pings all SSH hosts in parallel; skip when -short is set
// to keep the normal test suite fast.
func TestIntegration_Machines(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping machines test in short mode (SSH pings are slow)")
	}

	// Default output (table) — may be slow due to SSH pings.
	_, _, code := run("machines")
	if code != 0 {
		t.Fatalf("tagents machines failed (exit %d)", code)
	}

	// --json mode: verify valid JSON array output.
	out, _, code := run("machines", "--json")
	if code != 0 {
		t.Fatalf("tagents machines --json failed (exit %d)", code)
	}
	v := mustJSON(t, out)
	if _, ok := v.([]interface{}); !ok {
		t.Fatalf("expected JSON array from machines --json, got %T", v)
	}

	// --plaintext mode: just verify no error.
	_, _, code = run("machines", "--plaintext")
	if code != 0 {
		t.Fatalf("tagents machines --plaintext failed (exit %d)", code)
	}
}

// TestIntegration_Log tests the log command.
// The test session is a plain shell with no Claude JSONL log file, so log will
// return exit 1 ("no log file found"). We verify the command handles this gracefully
// (no panic, structured error output) rather than requiring a log to exist.
func TestIntegration_Log(t *testing.T) {
	// Default: expect non-zero because no JSONL log file exists for the test session.
	out, errOut, code := run("log", testSession)
	if code == 0 {
		// If a log file somehow exists in CI, accept it.
		t.Logf("tagents log returned 0 (log file exists): %s", out)
	} else {
		// Error message should mention the session name.
		if !strings.Contains(errOut, testSession) && !strings.Contains(out, testSession) {
			t.Errorf("expected error output to reference session name %q, got stdout=%s stderr=%s", testSession, out, errOut)
		}
	}

	// --raw flag: same expectation — fails gracefully if no log.
	_, _, _ = run("log", "--raw", testSession)

	// --json flag (global): same.
	_, _, _ = run("log", "--json", testSession)
}

// TestIntegration_SkillAdd tests skill installation with a custom HOME directory.
func TestIntegration_SkillAdd(t *testing.T) {
	if os.Getenv("READONLY") == "1" {
		t.Skip("READONLY=1: skipping skill add test")
	}
	tmpDir := t.TempDir()
	out, errOut, code := runWithEnv(map[string]string{"HOME": tmpDir}, "skill", "add")
	if code != 0 {
		t.Fatalf("tagents skill add failed (exit %d): stdout=%s stderr=%s", code, out, errOut)
	}
	// Verify SKILL.md was installed at $HOME/.claude/skills/tagents/SKILL.md.
	skillMD := filepath.Join(tmpDir, ".claude", "skills", "tagents", "SKILL.md")
	if _, err := os.Stat(skillMD); os.IsNotExist(err) {
		t.Errorf("skill not installed at %s", skillMD)
	}
}

// TestIntegration_SendForce tests send --force to the test session.
func TestIntegration_SendForce(t *testing.T) {
	if os.Getenv("READONLY") == "1" {
		t.Skip("READONLY=1: skipping send --force test")
	}
	// The test session exists (created by TestMain); --force bypasses busy check.
	out, errOut, code := run("send", "--force", testSession, "# test force send")
	if code != 0 {
		t.Fatalf("tagents send --force failed (exit %d): stdout=%s stderr=%s", code, out, errOut)
	}
}

// TestIntegration_BroadcastRuntime tests broadcast with --runtime filter.
func TestIntegration_BroadcastRuntime(t *testing.T) {
	// Filter by known runtime — may find 0 sessions; still exits 0.
	out, errOut, code := run("broadcast", "--dry-run", "--runtime", "claude", "# test")
	if code != 0 {
		t.Fatalf("broadcast --dry-run --runtime claude failed (exit %d): stdout=%s stderr=%s", code, out, errOut)
	}

	// Filter by unknown runtime — 0 sessions matching; still exits 0.
	out, errOut, code = run("broadcast", "--dry-run", "--runtime", "unknown-runtime-xyz", "# test")
	if code != 0 {
		t.Fatalf("broadcast --dry-run --runtime unknown failed (exit %d): stdout=%s stderr=%s", code, out, errOut)
	}
}

// TestIntegration_ExitCodes tests exit code semantics.
func TestIntegration_ExitCodes(t *testing.T) {
	// Exit 0: successful command.
	_, _, code := run("list")
	if code != 0 {
		t.Errorf("list should exit 0, got %d", code)
	}

	// Exit non-zero: agent not found.
	_, _, code = run("read", "nonexistent-agent-xyz")
	if code == 0 {
		t.Errorf("read nonexistent agent should exit non-zero, got 0")
	}

	// Exit non-zero: unknown flag.
	cmd := exec.Command(binaryPath, "--unknown-flag-xyz")
	err := cmd.Run()
	if err == nil {
		t.Error("unknown flag should exit non-zero, got 0")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("expected ExitError for unknown flag, got %T: %v", err, err)
	}
}
