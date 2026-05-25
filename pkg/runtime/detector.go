package runtime

import (
	"strings"
)

// DetectRuntime determines the agent runtime from session name and pane content.
// Heuristics are intentionally simple — false positives prefer "unknown" over wrong guesses.
func DetectRuntime(sessionName, paneContent string) Runtime {
	nameLower := strings.ToLower(sessionName)

	// Name-based: claude sessions often contain 'claude' or 'cc' prefix
	if strings.Contains(nameLower, "claude") {
		return Claude
	}
	// Common naming patterns: claude-main, cc-worker, agent-claude-1
	if strings.HasPrefix(nameLower, "cc-") || strings.HasSuffix(nameLower, "-cc") {
		return Claude
	}

	// Name-based: codex sessions
	if strings.Contains(nameLower, "codex") {
		return Codex
	}

	// Content-based: look for Claude Code prompt patterns
	if hasClaudioContent(paneContent) {
		return Claude
	}

	// Content-based: Codex patterns
	if hasCodexContent(paneContent) {
		return Codex
	}

	return Unknown
}

func hasClaudioContent(content string) bool {
	patterns := []string{
		"Human:",        // Claude conversation format
		"Assistant:",    // Claude conversation format
		"claude-sonnet", // Model name in output
		"claude-opus",   // Model name in output
		"claude-haiku",  // Model name in output
		"Claude Code",   // Self-identification
		"✓ Done",        // Claude Code completion marker
		"⎿",             // Claude Code tool use indicator
	}
	contentLower := strings.ToLower(content)
	for _, p := range patterns {
		if strings.Contains(contentLower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func hasCodexContent(content string) bool {
	patterns := []string{
		"codex",
		"openai",
	}
	contentLower := strings.ToLower(content)
	for _, p := range patterns {
		if strings.Contains(contentLower, p) {
			return true
		}
	}
	return false
}
