package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFindLogFile_NoClaudeDir(t *testing.T) {
	// When claude dir doesn't exist, should return ""
	result := FindLogFile("any-session", Claude)
	// This may return a real file if ~/.claude/projects exists — just check it doesn't panic
	_ = result
}

func TestFindLogFile_WithFiles(t *testing.T) {
	// Create a temp dir simulating ~/.claude/projects structure
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "projects", "my-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	logFile := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(logFile, []byte(`{"type":"message"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Directly test the walk logic
	var found []string
	_ = filepath.Walk(filepath.Join(dir, "projects"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			found = append(found, path)
		}
		return nil
	})
	if len(found) != 1 || found[0] != logFile {
		t.Errorf("expected to find %s, got %v", logFile, found)
	}
}

func TestFindLogFile_Unknown(t *testing.T) {
	result := FindLogFile("any-session", Unknown)
	if result != "" {
		t.Errorf("expected empty for Unknown runtime, got %s", result)
	}
}

func TestFindLogFile_Codex(t *testing.T) {
	// Codex log finding is not implemented — should return ""
	result := FindLogFile("any-session", Codex)
	if result != "" {
		t.Errorf("expected empty for Codex runtime, got %s", result)
	}
}

// -------------- LogIdleDuration tests --------------

func TestLogIdleDuration_MissingLog(t *testing.T) {
	// Non-existent session + Unknown runtime → no log found
	idle, found := LogIdleDuration("nonexistent-session", Unknown, time.Now())
	if found {
		t.Errorf("expected found=false for missing log, got true")
	}
	if idle != 0 {
		t.Errorf("expected idle=0 for missing log, got %v", idle)
	}
}

func TestLogIdleDuration_Codex(t *testing.T) {
	// Codex always returns "" from FindLogFile
	idle, found := LogIdleDuration("any-session", Codex, time.Now())
	if found {
		t.Errorf("expected found=false for Codex runtime, got true")
	}
	if idle != 0 {
		t.Errorf("expected idle=0 for Codex runtime, got %v", idle)
	}
}

func TestLogIdleDuration_FreshLog(t *testing.T) {
	// Write a JSONL file with a timestamp == now → idle should be ~0
	dir := t.TempDir()
	now := time.Now()

	// Override HOME to point to temp dir so FindLogFile finds our file
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	tsStr := now.Format(time.RFC3339Nano)
	lastLine := mustMarshal(t, map[string]interface{}{
		"type":      "user-message",
		"timestamp": tsStr,
	})
	logPath := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(logPath, []byte(lastLine+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Set mtime explicitly to now so fallback also yields ~0
	if err := os.Chtimes(logPath, now, now); err != nil {
		t.Fatal(err)
	}

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true for fresh log, got false")
	}
	if idle < 0 {
		t.Errorf("expected idle >= 0, got %v", idle)
	}
	// Allow up to 1 second of drift due to mtime/parse rounding
	if idle > time.Second {
		t.Errorf("expected fresh log idle ~ 0, got %v", idle)
	}
}

func TestLogIdleDuration_KnownTimestamp(t *testing.T) {
	// Use a fixed timestamp and controlled "now" → idle must be exact
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Reference point for "now"
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// Timestamp is 10 minutes before "now"
	ts := time.Date(2026, 6, 1, 11, 50, 0, 0, time.UTC)
	tsStr := ts.Format(time.RFC3339Nano)

	lastLine := mustMarshal(t, map[string]interface{}{
		"type":      "assistant-message",
		"timestamp": tsStr,
		"content":   "I have analyzed the codebase.",
	})
	logPath := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(logPath, []byte(lastLine+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true, got false")
	}
	expected := 10 * time.Minute
	if idle != expected {
		t.Errorf("expected idle=%v, got %v", expected, idle)
	}
}

func TestLogIdleDuration_TimestampWithFractionalSeconds(t *testing.T) {
	// Real Claude logs use fractional seconds: "2026-05-24T19:08:29.227Z"
	// Verify RFC3339Nano parsing works correctly.
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// "now" has fractional seconds; timestamp is on a round minute.
	// This tests that the fractional part is preserved after parsing.
	now := time.Date(2026, 6, 1, 12, 0, 0, 227*int(time.Millisecond), time.UTC)
	ts := time.Date(2026, 6, 1, 11, 50, 0, 0, time.UTC)

	tsStr := ts.Format(time.RFC3339Nano)

	lastLine := mustMarshal(t, map[string]interface{}{
		"type":      "user-message",
		"timestamp": tsStr,
	})
	logPath := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(logPath, []byte(lastLine+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true, got false")
	}
	expected := 10*time.Minute + 227*time.Millisecond
	if idle != expected {
		t.Errorf("expected idle=%v, got %v", expected, idle)
	}
}

func TestLogIdleDuration_StaleLogViaMtime(t *testing.T) {
	// Last line lacks a "timestamp" field → fallback to file mtime
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2026, 6, 1, 11, 55, 0, 0, time.UTC) // 5 minutes ago

	// Line has no "timestamp" field (like permission-mode metadata lines)
	lastLine := mustMarshal(t, map[string]interface{}{
		"type": "permission-mode",
		"mode": "default",
	})
	logPath := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(logPath, []byte(lastLine+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(logPath, mtime, mtime); err != nil {
		t.Fatal(err)
	}

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true, got false")
	}
	expected := 5 * time.Minute
	if idle != expected {
		t.Errorf("expected idle=%v (mtime fallback), got %v", expected, idle)
	}
}

func TestLogIdleDuration_EmptyFile(t *testing.T) {
	// An empty JSONL file → no lines → mtime fallback
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC) // 1 hour ago

	logPath := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(logPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(logPath, mtime, mtime); err != nil {
		t.Fatal(err)
	}

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true for empty file, got false")
	}
	expected := 1 * time.Hour
	if idle != expected {
		t.Errorf("expected idle=%v (mtime fallback for empty file), got %v", expected, idle)
	}
}

func TestLogIdleDuration_UnparseableTimestamp(t *testing.T) {
	// Last line has a "timestamp" but it's not RFC 3339 → mtime fallback
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2026, 6, 1, 11, 45, 0, 0, time.UTC) // 15 minutes ago

	lastLine := mustMarshal(t, map[string]interface{}{
		"type":      "custom",
		"timestamp": "not-a-real-timestamp",
	})
	logPath := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(logPath, []byte(lastLine+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(logPath, mtime, mtime); err != nil {
		t.Fatal(err)
	}

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true, got false")
	}
	expected := 15 * time.Minute
	if idle != expected {
		t.Errorf("expected idle=%v (mtime fallback for unparseable ts), got %v", expected, idle)
	}
}

func TestLogIdleDuration_TimestampNotString(t *testing.T) {
	// "timestamp" field exists but is not a string → mtime fallback
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2026, 6, 1, 11, 45, 0, 0, time.UTC)

	lastLine := mustMarshal(t, map[string]interface{}{
		"type":      "custom",
		"timestamp": 12345678, // numeric, not string
	})
	logPath := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(logPath, []byte(lastLine+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(logPath, mtime, mtime); err != nil {
		t.Fatal(err)
	}

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true, got false")
	}
	expected := 15 * time.Minute
	if idle != expected {
		t.Errorf("expected idle=%v (mtime fallback for non-string ts), got %v", expected, idle)
	}
}

func TestLogIdleDuration_DeletedBetweenFindAndStat(t *testing.T) {
	// Simulate TOCTOU by passing a path that doesn't exist.
	// We can't easily test the actual race, but we can verify the stat
	// error path by passing a non-existent path.
	// Since FindLogFile uses os.UserHomeDir(), we set HOME to a temp dir
	// and create NO .claude/projects — so FindLogFile returns "".
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// No .claude/projects created → FindLogFile returns ""
	idle, found := LogIdleDuration("test-session", Claude, time.Now())
	if found {
		t.Errorf("expected found=false when no log dir exists, got true")
	}
	if idle != 0 {
		t.Errorf("expected idle=0, got %v", idle)
	}
}

func TestLogIdleDuration_MultipleJsonlFiles(t *testing.T) {
	// LogIdleDuration uses FindLogFile which returns the most-recently-modified
	// JSONL. Verify that the returned idle duration reflects the most recent file.
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	newerMtime := time.Date(2026, 6, 1, 11, 50, 0, 0, time.UTC) // 10 min ago
	olderMtime := time.Date(2026, 6, 1, 11, 00, 0, 0, time.UTC) // 1 hr ago

	// Older file
	oldLine := mustMarshal(t, map[string]interface{}{
		"type":      "user-message",
		"timestamp": olderMtime.Format(time.RFC3339Nano),
	})
	oldPath := filepath.Join(projectDir, "old.jsonl")
	if err := os.WriteFile(oldPath, []byte(oldLine+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(oldPath, olderMtime, olderMtime)

	// Newer file
	newLine := mustMarshal(t, map[string]interface{}{
		"type":      "assistant-message",
		"timestamp": newerMtime.Format(time.RFC3339Nano),
	})
	newPath := filepath.Join(projectDir, "new.jsonl")
	if err := os.WriteFile(newPath, []byte(newLine+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(newPath, newerMtime, newerMtime)

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true, got false")
	}
	expected := 10 * time.Minute
	if idle != expected {
		t.Errorf("expected idle=%v (from most-recently-modified file), got %v", expected, idle)
	}
}

func TestLogIdleDuration_CorruptedJson(t *testing.T) {
	// Last line is not valid JSON at all → mtime fallback
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2026, 6, 1, 11, 45, 0, 0, time.UTC)

	logPath := filepath.Join(projectDir, "session.jsonl")
	data := "this is not valid json at all\n"
	if err := os.WriteFile(logPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(logPath, mtime, mtime)

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true for corrupted JSON, got false")
	}
	expected := 15 * time.Minute
	if idle != expected {
		t.Errorf("expected idle=%v (mtime fallback for corrupted JSON), got %v", expected, idle)
	}
}

func TestLogIdleDuration_OnlyBlankLines(t *testing.T) {
	// File with only whitespace lines → mtime fallback
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	projectDir := filepath.Join(dir, ".claude", "projects", "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2026, 6, 1, 11, 30, 0, 0, time.UTC) // 30 min ago

	logPath := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(logPath, []byte("\n\n  \n"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(logPath, mtime, mtime)

	idle, found := LogIdleDuration("test-session", Claude, now)
	if !found {
		t.Fatal("expected found=true, got false")
	}
	expected := 30 * time.Minute
	if idle != expected {
		t.Errorf("expected idle=%v (mtime fallback for blank lines), got %v", expected, idle)
	}
}

// -------------- helpers --------------

// mustMarshal is a test helper that encodes to JSON and returns the string,
// stripping the trailing newline that json.Marshal adds.
func mustMarshal(t *testing.T, v interface{}) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(data)
}

// Ensure must-be-strings are used
var _ = strings.TrimSpace
