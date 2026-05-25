package runtime

import "testing"

func TestDetectStatus(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    Status
	}{
		{"bash prompt", "some output\n$ ", Idle},
		{"zsh prompt", "some output\n% ", Idle},
		{"oh-my-zsh prompt", "output\n❯ ", Idle},
		{"root prompt", "output\n# ", Idle},
		{"bare dollar", "output\n$", Idle},
		{"bare percent", "output\n%", Idle},
		{"bare arrow", "output\n❯", Idle},
		{"busy running", "Analyzing codebase...\nChecking files...", Busy},
		{"empty content", "", Dead},
		{"whitespace only", "   \n   \n   ", Dead},
		{"claude thinking", "I'll analyze this step by step\nChecking the code...", Busy},
		{"multi-line with prompt at end", "line1\nline2\nresult\n$ ", Idle},
		{"claude human prompt at end", "some output\nHuman:", Idle},
		{"generic gt prompt", "output\n> ", Idle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectStatus(tt.content)
			if got != tt.want {
				t.Errorf("DetectStatus(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}
