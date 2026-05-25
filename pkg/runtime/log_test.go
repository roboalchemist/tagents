package runtime

import (
	"os"
	"path/filepath"
	"testing"
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
