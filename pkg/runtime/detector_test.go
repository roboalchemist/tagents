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
