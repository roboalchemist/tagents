package runtime

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FindLogFile finds the most recent log file for a session.
// For Claude Code: scans ~/.claude/projects/ for JSONL files.
// Returns "" if not found (not an error — log may not exist yet).
func FindLogFile(sessionName string, rt Runtime) string {
	switch rt {
	case Claude:
		return findClaudeLog(sessionName)
	case Codex:
		return findCodexLog(sessionName)
	default:
		return ""
	}
}

func findClaudeLog(sessionName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	claudeDir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return ""
	}

	// Find all JSONL files under ~/.claude/projects/
	var jsonlFiles []string
	_ = filepath.Walk(claudeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".jsonl") {
			jsonlFiles = append(jsonlFiles, path)
		}
		return nil
	})

	if len(jsonlFiles) == 0 {
		return ""
	}

	// Return the most recently modified JSONL file
	// (Better heuristic: find by CWD match, but session name isn't reliable for path matching)
	sort.Slice(jsonlFiles, func(i, j int) bool {
		iInfo, _ := os.Stat(jsonlFiles[i])
		jInfo, _ := os.Stat(jsonlFiles[j])
		if iInfo == nil || jInfo == nil {
			return false
		}
		return iInfo.ModTime().After(jInfo.ModTime())
	})
	return jsonlFiles[0]
}

func findCodexLog(sessionName string) string {
	// Codex log path conventions vary — return empty for now
	return ""
}
