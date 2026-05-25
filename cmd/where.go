package cmd

import (
	"fmt"

	"github.com/roboalchemist/tagents/pkg/output"
	"github.com/roboalchemist/tagents/pkg/tmux"
	"github.com/spf13/cobra"
)

var whereCmd = &cobra.Command{
	Use:   "where <agent>",
	Short: "Get agent's current working directory",
	Long: `Print the current working directory of the agent's tmux pane.

Examples:
  tagents where my-agent
  tagents where gateway:worker`,
	Example: `  tagents where my-agent
  tagents where gateway:worker`,
	Args: cobra.ExactArgs(1),
	RunE: runWhere,
}

func init() { rootCmd.AddCommand(whereCmd) }

func runWhere(cmd *cobra.Command, args []string) error {
	opts := GetOutputOptions()

	s, err := findSession(args[0])
	if err != nil {
		return err
	}

	client := tmux.NewLocalClient()
	cwd, err := client.GetPaneCWD(s.Name)
	if err != nil {
		return fmt.Errorf("getting CWD for %q: %w", s.Name, err)
	}

	if opts.Mode == output.ModeJSON {
		type cwdResult struct {
			CWD string `json:"cwd"`
		}
		return output.RenderJSON(cwdResult{CWD: cwd}, opts)
	}

	fmt.Println(cwd)
	return nil
}
