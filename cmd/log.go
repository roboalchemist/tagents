package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/roboalchemist/tagents/pkg/output"
	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/spf13/cobra"
)

var flagLogRaw bool

var logCmd = &cobra.Command{
	Use:   "log <agent> [lines]",
	Short: "Read agent session log",
	Long: `Read the agent's session transcript log.

For Claude Code agents: finds the most recent JSONL log file under ~/.claude/projects/
and renders it as human-readable conversation (or raw JSONL with --raw).

Default: last 100 lines.

Examples:
  tagents log my-agent
  tagents log my-agent 200
  tagents log my-agent --raw`,
	Example: `  tagents log my-agent
  tagents log my-agent 200 --raw`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runLog,
}

func init() {
	logCmd.Flags().BoolVar(&flagLogRaw, "raw", false, "Output raw JSONL without parsing")
	rootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, args []string) error {
	opts := GetOutputOptions()
	query := args[0]
	lines := 100
	if len(args) == 2 {
		var err error
		lines, err = strconv.Atoi(args[1])
		if err != nil || lines < 1 {
			return fmt.Errorf("invalid lines count %q", args[1])
		}
	}

	s, err := findSession(query)
	if err != nil {
		return err
	}

	logFile := runtime.FindLogFile(s.Name, s.Runtime)
	if logFile == "" {
		return fmt.Errorf("no log file found for agent %q (runtime: %s)", s.Name, s.Runtime)
	}

	f, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("opening log %s: %w", logFile, err)
	}
	defer f.Close()

	// Read all lines, keep last N
	var allLines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large JSONL lines
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading log: %w", err)
	}

	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}
	selected := allLines[start:]

	if flagLogRaw || opts.Mode == output.ModeJSON {
		for _, line := range selected {
			fmt.Println(line)
		}
		return nil
	}

	// Parse JSONL and render as conversation
	for _, line := range selected {
		if line == "" {
			continue
		}
		rendered := renderJSONLLine(line)
		if rendered != "" {
			fmt.Println(rendered)
		}
	}
	return nil
}

// renderJSONLLine parses a Claude Code JSONL line and returns a human-readable string.
// Returns "" for lines that should be skipped (system messages, tool results, etc.)
func renderJSONLLine(line string) string {
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return line // not JSON — print raw
	}

	msgType, _ := entry["type"].(string)
	role, _ := entry["role"].(string)

	switch msgType {
	case "user", "human":
		if content, ok := entry["content"].(string); ok {
			return fmt.Sprintf("Human: %s", content)
		}
	case "assistant":
		if content, ok := entry["content"].(string); ok {
			return fmt.Sprintf("Assistant: %s", truncate(content, 200))
		}
	case "message":
		if role == "user" || role == "human" {
			if content, ok := entry["content"].(string); ok {
				return fmt.Sprintf("Human: %s", content)
			}
		}
		if role == "assistant" {
			if content, ok := entry["content"].(string); ok {
				return fmt.Sprintf("Assistant: %s", truncate(content, 200))
			}
		}
	}

	// Skip tool calls/results (too noisy)
	if strings.Contains(line, "tool_use") || strings.Contains(line, "tool_result") {
		return ""
	}

	return ""
}
