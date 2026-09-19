// Package launch builds the shell command line used to start an agent harness
// inside a freshly created tmux session.
package launch

import (
	"fmt"
	"strings"
)

// Canonical harness names accepted by Spec.Harness.
const (
	OpenCode = "opencode"
	Claude   = "claude"
	Codex    = "codex"
	Pi       = "pi"
)

// Spec describes how to launch an agent.
type Spec struct {
	Harness string // opencode | claude | codex | pi; empty defaults to opencode
	Model   string // optional model id, e.g. "haiku[1m]" or "claude-sonnet-4"
	Command string // optional explicit command; when set it wins over Harness/Model
}

// Command returns the shell command line that starts the agent. The line is
// intended to be typed into an interactive shell, so the model argument is
// quoted to survive glob/shell expansion (e.g. "haiku[1m]").
func (s Spec) CommandLine() (string, error) {
	if cmd := strings.TrimSpace(s.Command); cmd != "" {
		return cmd, nil
	}

	rt, err := NormalizeHarness(s.Harness)
	if err != nil {
		return "", err
	}

	switch rt {
	case OpenCode:
		return withModel("opencode", s.Model, "--model"), nil
	case Claude:
		return withModel("claude", s.Model, "--model"), nil
	case Codex:
		return withModel("codex", s.Model, "-m"), nil
	case Pi:
		return withModel("pi", s.Model, "--model"), nil
	}
	// NormalizeHarness only returns the four names above, so this is unreachable.
	return "", fmt.Errorf("unsupported runtime %q", rt)
}

// NormalizeHarness maps a user-supplied harness name to its canonical form.
// An empty value defaults to opencode. "oc" is accepted as an alias.
func NormalizeHarness(r string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(r)) {
	case "", OpenCode, "oc":
		return OpenCode, nil
	case Claude:
		return Claude, nil
	case Codex:
		return Codex, nil
	case Pi:
		return Pi, nil
	}
	return "", fmt.Errorf("unknown runtime %q (want opencode, claude, codex, or pi; or pass --command)", r)
}

func withModel(base, model, flag string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return base
	}
	return base + " " + flag + " " + quote(model)
}

// quote single-quotes s when it contains characters the shell would otherwise
// expand. Safe strings pass through unchanged for readability.
func quote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`{}()[]|&;<>*?!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
