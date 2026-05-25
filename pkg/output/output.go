package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// Mode controls output format.
type Mode int

const (
	ModeTable     Mode = iota
	ModeJSON
	ModePlaintext
)

// Options configures output rendering.
type Options struct {
	Mode    Mode
	NoColor bool
	Debug   bool
	Fields  string // comma-separated field selection
	JQ      string // JQ filter expression
}

// ShouldUseColor returns true if color output is enabled.
func (o Options) ShouldUseColor() bool {
	if o.NoColor {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Auto-detect TTY: if stdout is not a terminal, disable color
	fi, err := os.Stdout.Stat()
	if err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		return false
	}
	return true
}

// RenderJSON marshals v to indented JSON and prints to stdout.
// If opts.JQ is set, applies the JQ filter.
// If opts.Fields is set, prunes to selected top-level fields.
func RenderJSON(v interface{}, opts Options) error {
	// Apply field selection if requested
	if opts.Fields != "" {
		v = pruneFields(v, splitFields(opts.Fields))
	}

	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON marshal: %w", err)
	}

	// Apply JQ filter if requested
	if opts.JQ != "" {
		filtered, err := applyJQ(b, opts.JQ)
		if err != nil {
			return err
		}
		fmt.Println(filtered)
		return nil
	}

	fmt.Println(string(b))
	return nil
}

// RenderTable prints a table to stdout.
func RenderTable(headers []string, rows [][]string, opts Options) error {
	if !opts.ShouldUseColor() {
		color.NoColor = true
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("  ")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	table.SetAutoWrapText(false)
	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
	return nil
}

// RenderPlaintext prints tab-separated output with a header row.
func RenderPlaintext(headers []string, rows [][]string, opts Options) error {
	fmt.Println(strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Println(strings.Join(row, "\t"))
	}
	return nil
}

// RenderError prints a structured error to stderr.
// In JSON mode, outputs {"error": "...", "code": N}.
// In other modes, prints "Error: ..." to stderr.
func RenderError(msg string, code int, opts Options) {
	if opts.Mode == ModeJSON {
		b, _ := json.Marshal(map[string]interface{}{"error": msg, "code": code})
		fmt.Fprintln(os.Stderr, string(b))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
}

// splitFields splits a comma-separated field list into a slice.
func splitFields(fields string) []string {
	if fields == "" {
		return nil
	}
	var result []string
	for _, f := range strings.Split(fields, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			result = append(result, f)
		}
	}
	return result
}

// pruneFields restricts a JSON-marshallable value to only the named fields.
// Works on maps and slices of maps. Other types are returned unchanged.
func pruneFields(v interface{}, fields []string) interface{} {
	if len(fields) == 0 {
		return v
	}
	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[strings.ToLower(f)] = true
	}
	return pruneFieldsResolved(v, fieldSet)
}

// pruneFieldsResolved prunes fields from a JSON-native or arbitrary struct value.
// Arbitrary structs are round-tripped to JSON-native types exactly once.
func pruneFieldsResolved(v interface{}, fieldSet map[string]bool) interface{} {
	switch val := v.(type) {
	case []map[string]interface{}:
		result := make([]map[string]interface{}, len(val))
		for i, m := range val {
			result[i] = pruneMap(m, fieldSet)
		}
		return result
	case []interface{}:
		// Slice of arbitrary values (e.g. after JSON unmarshal of []SomeStruct)
		result := make([]interface{}, len(val))
		for i, item := range val {
			if m, ok := item.(map[string]interface{}); ok {
				result[i] = pruneMap(m, fieldSet)
			} else {
				result[i] = item
			}
		}
		return result
	case map[string]interface{}:
		return pruneMap(val, fieldSet)
	default:
		// For arbitrary structs, round-trip through JSON once to get native types.
		b, err := json.Marshal(v)
		if err != nil {
			return v
		}
		var m interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return v
		}
		// m is now a JSON-native type; recurse without risk of infinite loop.
		return pruneFieldsResolved(m, fieldSet)
	}
}

func pruneMap(m map[string]interface{}, keep map[string]bool) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		if keep[strings.ToLower(k)] {
			result[k] = v
		}
	}
	return result
}
