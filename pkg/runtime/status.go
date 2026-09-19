package runtime

import (
	"strings"
)

// DetectStatus determines agent status from the last N lines of pane content.
// Logic: if the last non-empty line ends with a shell prompt character, the agent is idle.
func DetectStatus(paneContent string) Status {
	if strings.TrimSpace(paneContent) == "" {
		return Dead
	}

	// Get the last meaningful line
	lines := strings.Split(paneContent, "\n")
	lastLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" {
			lastLine = trimmed
			break
		}
	}

	if lastLine == "" {
		return Dead
	}

	// Shell prompt patterns — if the last line ends with one of these, agent is waiting
	promptSuffixes := []string{
		"$ ", // bash/sh
		"% ", // zsh
		"❯ ", // oh-my-zsh and common prompt themes
		"> ", // generic
		"# ", // root shell
		"$ ", // trailing space variants
	}
	for _, suffix := range promptSuffixes {
		if strings.HasSuffix(lastLine, strings.TrimRight(suffix, " ")) ||
			strings.HasSuffix(lastLine, suffix) {
			return Idle
		}
	}

	// Also check if the last line IS just a prompt symbol
	promptExact := []string{"$", "%", "❯", ">", "#"}
	for _, p := range promptExact {
		if lastLine == p {
			return Idle
		}
	}

	// Lines that indicate claude is waiting for input
	claudeWaiting := []string{
		"Human:",
		"> ", // Claude's input prompt
	}
	for _, p := range claudeWaiting {
		if strings.HasSuffix(lastLine, strings.TrimRight(p, " ")) {
			return Idle
		}
	}

	return Busy
}

// DetectStatusWithRuntime determines status using runtime-aware signals.
// OpenCode's TUI keeps a status bar as the last visible line, so shell-prompt
// heuristics never fire; it instead exposes "esc interrupt" only while a turn
// is running. All other runtimes fall back to DetectStatus.
func DetectStatusWithRuntime(paneContent string, rt Runtime) Status {
	if rt == OpenCode {
		return detectOpenCodeStatus(paneContent)
	}
	return DetectStatus(paneContent)
}

func detectOpenCodeStatus(content string) Status {
	if strings.TrimSpace(content) == "" {
		return Dead
	}
	lower := strings.ToLower(content)
	// Only trust the heuristic when the OpenCode TUI is actually on screen.
	if !strings.Contains(lower, "ctrl+p commands") && !strings.Contains(lower, "• opencode") {
		return DetectStatus(content)
	}
	// "esc interrupt" is shown for the duration of an active turn.
	if strings.Contains(lower, "esc interrupt") {
		return Busy
	}
	return Idle
}
