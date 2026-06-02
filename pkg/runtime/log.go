package runtime

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

// LogIdleDuration determines how long an agent's session log has been idle.
// It locates the transcript via FindLogFile and returns the time since
// the log last advanced. Prefers parsing the last JSON line's "timestamp"
// field (RFC 3339 with fractional seconds); falls back to file mtime when
// the timestamp is missing or unparseable.
//
// found is false when no log file exists for this session+Runtime pair
// (e.g. Codex, Unknown, or no Claude project directory).
func LogIdleDuration(sessionName string, rt Runtime, now time.Time) (idle time.Duration, found bool) {
	return LogIdleDurationFromPath(FindLogFile(sessionName, rt), now)
}

// LogIdleDurationFromPath determines how long a known session log path has been idle.
// It is useful for callers that poll repeatedly and want to cache FindLogFile's
// filesystem walk outside the hot path.
func LogIdleDurationFromPath(logPath string, now time.Time) (idle time.Duration, found bool) {
	if logPath == "" {
		return 0, false
	}

	fi, err := os.Stat(logPath)
	if err != nil {
		// TOCTOU: file deleted between FindLogFile and Stat
		return 0, false
	}
	mtimeIdle := now.Sub(fi.ModTime())

	// Read tail of file to obtain the last JSON line.
	// 4 KiB is enough to capture the last line of even large tool-result
	// entries while keeping the read cheap.
	const tailSize = 4096
	f, err := os.Open(logPath)
	if err != nil {
		// Another TOCTOU — treat as not found
		return 0, false
	}
	defer f.Close()

	offset := fi.Size() - tailSize
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, 0); err != nil {
		return mtimeIdle, true
	}

	var lastLine string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lastLine = line
		}
	}

	if lastLine == "" {
		// Empty file or only blank lines — fall back to mtime
		return mtimeIdle, true
	}

	// Attempt to extract the "timestamp" field from the last JSON line
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(lastLine), &entry); err != nil {
		return mtimeIdle, true
	}

	tsRaw, ok := entry["timestamp"]
	if !ok {
		return mtimeIdle, true
	}
	tsStr, ok := tsRaw.(string)
	if !ok {
		return mtimeIdle, true
	}

	parsed, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		return mtimeIdle, true
	}

	return now.Sub(parsed), true
}
