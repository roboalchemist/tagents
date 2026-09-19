package launch

import "testing"

func TestSpec_Command(t *testing.T) {
	tests := []struct {
		name    string
		spec    Spec
		want    string
		wantErr bool
	}{
		{"default runtime is opencode", Spec{}, "opencode", false},
		{"explicit opencode", Spec{Runtime: "opencode"}, "opencode", false},
		{"oc alias", Spec{Runtime: "oc"}, "opencode", false},
		{"opencode with model is quoted", Spec{Runtime: "opencode", Model: "haiku[1m]"}, "opencode --model 'haiku[1m]'", false},
		{"opencode with plain model", Spec{Runtime: "opencode", Model: "gpt-5.6-sol"}, "opencode --model gpt-5.6-sol", false},
		{"claude with model", Spec{Runtime: "claude", Model: "claude-sonnet-4"}, "claude --model claude-sonnet-4", false},
		{"codex uses -m", Spec{Runtime: "codex", Model: "gpt-5.6-terra"}, "codex -m gpt-5.6-terra", false},
		{"pi without model", Spec{Runtime: "pi"}, "pi", false},
		{"explicit command wins", Spec{Runtime: "opencode", Model: "x", Command: "vcodex --app"}, "vcodex --app", false},
		{"explicit command is trimmed", Spec{Command: "  my-launcher  "}, "my-launcher", false},
		{"unknown runtime", Spec{Runtime: "bogus"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.spec.CommandLine()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Command() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeRuntime(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", OpenCode, false},
		{"opencode", OpenCode, false},
		{"OpenCode", OpenCode, false},
		{"OC", OpenCode, false},
		{"claude", Claude, false},
		{"codex", Codex, false},
		{"pi", Pi, false},
		{"clippy", "", true},
	}
	for _, tt := range tests {
		got, err := NormalizeRuntime(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NormalizeRuntime(%q) expected error, got %q", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeRuntime(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeRuntime(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestQuote(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"simple", "simple"},
		{"gpt-5.6-sol", "gpt-5.6-sol"},
		{"haiku[1m]", "'haiku[1m]'"},
		{"with space", "'with space'"},
		{"it's", `'it'\''s'`},
		{"", "''"},
	}
	for _, tt := range tests {
		if got := quote(tt.in); got != tt.want {
			t.Errorf("quote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
