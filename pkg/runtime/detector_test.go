package runtime

import (
	"testing"
)

func TestDetectRuntime_ByName(t *testing.T) {
	tests := []struct {
		name    string
		session string
		want    Runtime
	}{
		{"claude in name", "my-claude-session", Claude},
		{"cc prefix", "cc-worker-1", Claude},
		{"codex in name", "codex-agent", Codex},
		{"opencode in name", "my-opencode-session", OpenCode},
		{"oc prefix", "oc-worker-1", OpenCode},
		{"opencode wins over codex in mixed name", "codex-and-opencode", OpenCode},
		{"pi token", "pi-1", Pi},
		{"pi token suffix", "my-pi", Pi},
		{"pi bare", "pi", Pi},
		{"pi substring is not a token", "api", Unknown},
		{"pi substring inside word", "pipeline", Unknown},
		{"unknown name", "random-session", Unknown},
		{"empty", "", Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectRuntime(tt.session, "")
			if got != tt.want {
				t.Errorf("DetectRuntime(%q, \"\") = %v, want %v", tt.session, got, tt.want)
			}
		})
	}
}

func TestDetectRuntime_ByContent(t *testing.T) {
	tests := []struct {
		name    string
		session string
		content string
		want    Runtime
	}{
		{"claude tag in content", "session", "Human: fix this bug", Claude},
		{"claude assistant tag", "session", "Assistant: here is the fix", Claude},
		{"claude model name", "session", "Using claude-sonnet-4 model", Claude},
		{"claude checkmark", "session", "✓ Done\n❯ ", Claude},
		{"codex content", "session", "openai codex response", Codex},
		{"opencode status bar", "session", "  Build · haiku[1m]  ctrl+p commands   • OpenCode 1.18.31", OpenCode},
		{"opencode branding only", "session", "• OpenCode 1.18.31", OpenCode},
		{"pi startup header", "session", "pi v0.85.1\n/ commands · ! bash", Pi},
		{"pi rebrand header", "session", "π v0.85.1", Pi},
		{"pi onboarding text", "session", "Pi can explain its own features and look up its docs.", Pi},
		{"no markers", "session", "echo hello\n$ ", Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectRuntime(tt.session, tt.content)
			if got != tt.want {
				t.Errorf("DetectRuntime(%q, content) = %v, want %v", tt.session, got, tt.want)
			}
		})
	}
}

func TestDetectRuntime_NameTakesPrecedence(t *testing.T) {
	// Name match should return without checking content
	got := DetectRuntime("my-claude-session", "codex usage here")
	if got != Claude {
		t.Errorf("expected Claude (name match), got %v", got)
	}
}

func TestDetectRuntimeWithProcess_FromCommand(t *testing.T) {
	tests := []struct {
		name    string
		session string
		content string
		command string
		title   string
		want    Runtime
	}{
		// Process command is authoritative, even over misleading names/content.
		{"opencode command", "random", "Human: hi", "opencode", "", OpenCode},
		{"claude command over opencode content", "random", "ctrl+p commands", "claude", "", Claude},
		{"codex command", "random", "", "codex", "", Codex},
		{"vcodex wrapper resolves to opencode", "vcodex-doctor", "", "opencode", "", OpenCode},
		{"absolute path command", "random", "", "/usr/local/bin/claude", "", Claude},
		{"login shell is not a runtime", "random", "Human: hi", "-zsh", "", Claude},
		{"node with pi title", "random", "some output", "node", "π - myproject", Pi},
		{"node with opencode title", "random", "some output", "node", "OC | my task", OpenCode},
		{"interpreter falls back to name", "pi-1", "busy output", "node", "", Pi},
		{"shell falls back to content", "random", "ctrl+p commands", "zsh", "", OpenCode},
		{"nothing known", "random", "plain output", "zsh", "", Unknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectRuntimeWithProcess(tt.session, tt.content, tt.command, tt.title)
			if got != tt.want {
				t.Errorf("DetectRuntimeWithProcess(%q, content, %q, %q) = %v, want %v",
					tt.session, tt.command, tt.title, got, tt.want)
			}
		})
	}
}

func TestRuntimeFromTitle(t *testing.T) {
	tests := []struct {
		title string
		want  Runtime
	}{
		{"OC | Tagents pi support", OpenCode},
		{"OC | anything", OpenCode},
		{"π - tagents", Pi},
		{"pi - myproject", Pi},
		{"", Unknown},
		{"my shell", Unknown},
		{"Tagents pi support", Unknown}, // "pi" inside prose must not match
	}
	for _, tt := range tests {
		if got := runtimeFromTitle(tt.title); got != tt.want {
			t.Errorf("runtimeFromTitle(%q) = %v, want %v", tt.title, got, tt.want)
		}
	}
}

func TestRuntimeFromCommand(t *testing.T) {
	tests := []struct {
		command string
		want    Runtime
	}{
		{"claude", Claude},
		{"/opt/homebrew/bin/claude", Claude},
		{"codex", Codex},
		{"opencode", OpenCode},
		{"vcodex", OpenCode},
		{"vopencode", OpenCode},
		{"pi", Pi},
		{"-zsh", Unknown},
		{"node", Unknown},
		{"python3", Unknown},
		{"", Unknown},
	}
	for _, tt := range tests {
		if got := runtimeFromCommand(tt.command); got != tt.want {
			t.Errorf("runtimeFromCommand(%q) = %v, want %v", tt.command, got, tt.want)
		}
	}
}
