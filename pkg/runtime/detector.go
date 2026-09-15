package runtime

import (
	"strings"
)

// DetectRuntime determines the agent runtime from session name and pane content.
// Heuristics are intentionally simple — false positives prefer "unknown" over wrong guesses.
func DetectRuntime(sessionName, paneContent string) Runtime {
	return DetectRuntimeWithProcess(sessionName, paneContent, "", "")
}

// DetectRuntimeWithProcess determines the agent runtime using process-level
// signals first, then falling back to session name and pane content heuristics.
//
// command is the pane's foreground command (tmux #{pane_current_command}) and
// title is the pane title (tmux #{pane_title}). These are independent of which
// text is currently visible in the pane, so they are far more stable than
// inspecting pane content — an agent that fills the screen with output still
// reports the same foreground command.
func DetectRuntimeWithProcess(sessionName, paneContent, command, title string) Runtime {
	if rt := runtimeFromCommand(command); rt != Unknown {
		return rt
	}
	if rt := runtimeFromTitle(title); rt != Unknown {
		return rt
	}
	return detectRuntimeHeuristic(sessionName, paneContent)
}

// runtimeFromCommand maps a pane's foreground command to a runtime.
// Returns Unknown for shells and interpreters, which require further signals.
func runtimeFromCommand(command string) Runtime {
	c := strings.ToLower(strings.TrimSpace(command))
	c = strings.TrimPrefix(c, "-") // login shells report "-zsh"
	if i := strings.LastIndexByte(c, '/'); i >= 0 {
		c = c[i+1:]
	}
	switch c {
	case "claude":
		return Claude
	case "codex":
		return Codex
	case "opencode", "vcodex", "vopencode":
		return OpenCode
	case "pi":
		return Pi
	}
	return Unknown
}

// runtimeFromTitle maps an application-set pane title to a runtime. This covers
// agents that run under an interpreter (e.g. pi runs as "node") and therefore
// do not name themselves in pane_current_command.
func runtimeFromTitle(title string) Runtime {
	t := strings.TrimSpace(title)
	if t == "" {
		return Unknown
	}
	lower := strings.ToLower(t)
	// OpenCode sets the pane title to "OC | <session title>".
	if lower == "oc" || strings.HasPrefix(lower, "oc |") || strings.HasPrefix(lower, "oc|") {
		return OpenCode
	}
	// Pi sets the pane title to "π - <cwd>" ("<app> - <cwd>" for rebrands).
	if strings.HasPrefix(t, "π") || lower == "pi" || strings.HasPrefix(lower, "pi -") {
		return Pi
	}
	return Unknown
}

func detectRuntimeHeuristic(sessionName, paneContent string) Runtime {
	nameLower := strings.ToLower(sessionName)

	// Name-based: claude sessions often contain 'claude' or 'cc' prefix
	if strings.Contains(nameLower, "claude") {
		return Claude
	}
	// Common naming patterns: claude-main, cc-worker, agent-claude-1
	if strings.HasPrefix(nameLower, "cc-") || strings.HasSuffix(nameLower, "-cc") {
		return Claude
	}

	// Name-based: opencode sessions. Checked before codex so a mixed name like
	// "codex-and-opencode" resolves to opencode.
	if strings.Contains(nameLower, "opencode") {
		return OpenCode
	}
	if strings.HasPrefix(nameLower, "oc-") || strings.HasSuffix(nameLower, "-oc") {
		return OpenCode
	}

	// Name-based: codex sessions
	if strings.Contains(nameLower, "codex") {
		return Codex
	}

	// Name-based: pi sessions. Match "pi" as a whole token only — a plain
	// substring check would misfire on names like "api" or "pipeline".
	if hasNameToken(nameLower, "pi") {
		return Pi
	}

	// Content-based: OpenCode's persistent status bar is unambiguous.
	if hasOpenCodeContent(paneContent) {
		return OpenCode
	}

	// Content-based: look for Claude Code prompt patterns
	if hasClaudeContent(paneContent) {
		return Claude
	}

	// Content-based: Codex patterns
	if hasCodexContent(paneContent) {
		return Codex
	}

	// Content-based: pi startup header
	if hasPiContent(paneContent) {
		return Pi
	}

	return Unknown
}

// hasNameToken reports whether name contains word as a separator-delimited token
// (e.g. "pi" matches "pi-1" and "my-pi" but not "api" or "pipeline").
func hasNameToken(nameLower, word string) bool {
	separators := func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == ' ' || r == '/'
	}
	for _, token := range strings.FieldsFunc(nameLower, separators) {
		if token == word {
			return true
		}
	}
	return false
}

func hasOpenCodeContent(content string) bool {
	patterns := []string{
		"ctrl+p commands", // OpenCode status bar keybinding hint
		"• opencode",      // OpenCode status bar branding
	}
	contentLower := strings.ToLower(content)
	for _, p := range patterns {
		if strings.Contains(contentLower, p) {
			return true
		}
	}
	return false
}

func hasPiContent(content string) bool {
	contentLower := strings.ToLower(content)
	if strings.Contains(contentLower, "pi can explain its own features") ||
		strings.Contains(contentLower, "to show full startup help and loaded resources") {
		return true
	}
	// Startup header logo, e.g. "pi v0.85.1" (or "π v..." for rebrands).
	for _, line := range strings.Split(contentLower, "\n") {
		line = strings.TrimSpace(line)
		for _, prefix := range []string{"pi v", "π v"} {
			if rest, ok := strings.CutPrefix(line, prefix); ok && rest != "" && rest[0] >= '0' && rest[0] <= '9' {
				return true
			}
		}
	}
	return false
}

func hasClaudeContent(content string) bool {
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
