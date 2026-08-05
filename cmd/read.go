package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/roboalchemist/tagents/pkg/output"
	"github.com/roboalchemist/tagents/pkg/session"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <agent> [lines]",
	Short: "Read last N lines of agent's tmux pane output",
	Long: `Read the last N lines of an agent's tmux pane output.

Default: 50 lines. Agent name is fuzzy-matched.

Examples:
  tagents read my-agent
  tagents read my-agent 100
  tagents read my-agent --json
  tagents read gateway:worker 50`,
	Example: `  tagents read my-agent
  tagents read my-agent 100 --json`,
	SuggestFor: []string{"tail", "cat", "view"},
	Args:       cobra.RangeArgs(1, 2),
	RunE:       runRead,
}

func init() { rootCmd.AddCommand(readCmd) }

func runRead(cmd *cobra.Command, args []string) error {
	opts := GetOutputOptions()
	query := args[0]
	lines := 50
	if len(args) == 2 {
		var err error
		lines, err = strconv.Atoi(args[1])
		if err != nil || lines < 1 {
			return fmt.Errorf("invalid lines count %q: must be a positive integer", args[1])
		}
	}

	s, err := findSession(query)
	if err != nil {
		return err
	}

	client, err := session.ClientFor(*s, 10*time.Second)
	if err != nil {
		return err
	}
	content, err := client.CapturePane(s.Name, lines)
	if err != nil {
		return fmt.Errorf("reading pane for %q: %w", s.Name, err)
	}

	if opts.Mode == output.ModeJSON {
		type paneResult struct {
			Session string   `json:"session"`
			Machine string   `json:"machine"`
			Lines   []string `json:"lines"`
		}
		lineSlice := strings.Split(content, "\n")
		result := paneResult{
			Session: s.Name,
			Machine: s.Machine,
			Lines:   lineSlice,
		}
		return output.RenderJSON(result, opts)
	}

	fmt.Print(content)
	if !strings.HasSuffix(content, "\n") {
		fmt.Println()
	}
	return nil
}
