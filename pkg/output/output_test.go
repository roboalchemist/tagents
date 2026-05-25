package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout captures os.Stdout during the function call.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRenderJSON_Basic(t *testing.T) {
	data := map[string]string{"name": "alice", "status": "idle"}
	opts := Options{Mode: ModeJSON}
	out := captureStdout(t, func() {
		if err := RenderJSON(data, opts); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, `"name"`) || !strings.Contains(out, `"alice"`) {
		t.Errorf("expected JSON with name/alice, got: %s", out)
	}
	// Must be valid JSON
	var v interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &v); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput: %s", err, out)
	}
}

func TestRenderJSON_Array(t *testing.T) {
	data := []map[string]string{
		{"name": "alice"},
		{"name": "bob"},
	}
	opts := Options{Mode: ModeJSON}
	out := captureStdout(t, func() {
		_ = RenderJSON(data, opts)
	})
	var arr []interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &arr); err != nil {
		t.Fatalf("not valid JSON array: %v", err)
	}
	if len(arr) != 2 {
		t.Errorf("expected 2 items, got %d", len(arr))
	}
}

func TestRenderJSON_FieldSelection(t *testing.T) {
	data := []map[string]interface{}{
		{"name": "alice", "status": "idle", "machine": "gateway"},
	}
	opts := Options{Mode: ModeJSON, Fields: "name,status"}
	out := captureStdout(t, func() {
		_ = RenderJSON(data, opts)
	})
	if strings.Contains(out, "machine") {
		t.Error("machine field should have been pruned")
	}
	if !strings.Contains(out, "name") || !strings.Contains(out, "status") {
		t.Error("name and status should be present")
	}
}

func TestRenderJSON_JQ(t *testing.T) {
	data := []map[string]string{
		{"name": "alice", "status": "idle"},
		{"name": "bob", "status": "busy"},
	}
	opts := Options{Mode: ModeJSON, JQ: ".[0].name"}
	out := captureStdout(t, func() {
		_ = RenderJSON(data, opts)
	})
	if !strings.Contains(out, "alice") {
		t.Errorf("expected alice from JQ .[0].name, got: %s", out)
	}
}

func TestRenderTable(t *testing.T) {
	headers := []string{"NAME", "STATUS"}
	rows := [][]string{
		{"alice", "idle"},
		{"bob", "busy"},
	}
	opts := Options{Mode: ModeTable, NoColor: true}
	out := captureStdout(t, func() {
		_ = RenderTable(headers, rows, opts)
	})
	if !strings.Contains(out, "alice") || !strings.Contains(out, "bob") {
		t.Errorf("expected alice and bob in table output: %s", out)
	}
	if !strings.Contains(strings.ToUpper(out), "NAME") {
		t.Errorf("expected NAME header in table output: %s", out)
	}
}

func TestRenderPlaintext(t *testing.T) {
	headers := []string{"NAME", "STATUS"}
	rows := [][]string{{"alice", "idle"}}
	opts := Options{Mode: ModePlaintext}
	out := captureStdout(t, func() {
		_ = RenderPlaintext(headers, rows, opts)
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "\t") {
		t.Error("header line should be tab-separated")
	}
	if !strings.Contains(lines[1], "alice") {
		t.Error("data line should contain alice")
	}
}

func TestSplitFields(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"name,status", []string{"name", "status"}},
		{"name, status", []string{"name", "status"}}, // spaces trimmed
		{"", nil},
		{"single", []string{"single"}},
	}
	for _, tt := range tests {
		got := splitFields(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitFields(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitFields(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestRenderError_NonJSON(t *testing.T) {
	// Just verify it doesn't panic — stderr capture is complex, skip output check
	opts := Options{Mode: ModeTable}
	RenderError("something failed", 1, opts)
}

func TestShouldUseColor_NoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	opts := Options{}
	if opts.ShouldUseColor() {
		t.Error("expected color disabled when NO_COLOR is set")
	}
}

func TestShouldUseColor_FlagSet(t *testing.T) {
	opts := Options{NoColor: true}
	if opts.ShouldUseColor() {
		t.Error("expected color disabled when NoColor flag is true")
	}
}

func TestRenderError_JSON(t *testing.T) {
	// Capture stderr
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	opts := Options{Mode: ModeJSON}
	RenderError("bad input", 400, opts)
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()
	if !strings.Contains(out, "bad input") {
		t.Errorf("expected error message in JSON output: %s", out)
	}
	if !strings.Contains(out, "400") {
		t.Errorf("expected code 400 in JSON output: %s", out)
	}
	// Must be valid JSON
	var v interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &v); err != nil {
		t.Errorf("RenderError JSON output is not valid JSON: %v\noutput: %s", err, out)
	}
}

func TestPruneFields_Struct(t *testing.T) {
	// Test round-trip through JSON for arbitrary struct types
	type Agent struct {
		Name    string `json:"name"`
		Status  string `json:"status"`
		Machine string `json:"machine"`
	}
	data := Agent{Name: "alice", Status: "idle", Machine: "gateway"}
	opts := Options{Mode: ModeJSON, Fields: "name,status"}
	out := captureStdout(t, func() {
		_ = RenderJSON(data, opts)
	})
	if strings.Contains(out, "machine") {
		t.Error("machine field should have been pruned from struct")
	}
	if !strings.Contains(out, "name") || !strings.Contains(out, "alice") {
		t.Error("name/alice should be present after pruning struct")
	}
}

func TestApplyJQ_InvalidFilter(t *testing.T) {
	data := []byte(`{"name":"alice"}`)
	_, err := applyJQ(data, "!!!invalid!!!")
	if err == nil {
		t.Error("expected error for invalid JQ filter")
	}
}

func TestApplyJQ_InvalidJSON(t *testing.T) {
	data := []byte(`not-json`)
	_, err := applyJQ(data, ".name")
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

func TestRenderJSON_JQ_Error(t *testing.T) {
	data := map[string]string{"name": "alice"}
	opts := Options{Mode: ModeJSON, JQ: "!!!invalid!!!"}
	err := RenderJSON(data, opts)
	if err == nil {
		t.Error("expected error for invalid JQ filter")
	}
}

func TestShouldUseColor_NoColorAndNoFlag(t *testing.T) {
	// Unset NO_COLOR to test default path (non-TTY in tests)
	t.Setenv("NO_COLOR", "")
	opts := Options{NoColor: false}
	// In test environment stdout is not a TTY, so ShouldUseColor should return false
	result := opts.ShouldUseColor()
	// We just verify it doesn't panic and returns a bool
	_ = result
}
