package cmd

import (
	"fmt"
	"os"

	"github.com/roboalchemist/tagents/pkg/output"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agent sessions",
	Long: `List all agent sessions in scope.

Columns: machine (only when multi-machine scope), session name, runtime, status, cwd, preview.
Status colors: idle=green, busy=yellow, dead=red.

Examples:
  tagents list
  tagents list --json
  tagents list --machine gateway
  tagents list --all-machines
  tagents list --json --fields name,status`,
	Example: `  tagents list
  tagents list --json | jq '.[0]'
  tagents list --all-machines --plaintext`,
	SuggestFor: []string{"ls", "ps"},
	RunE:       runList,
}

func init() { rootCmd.AddCommand(listCmd) }

func runList(cmd *cobra.Command, args []string) error {
	opts := GetOutputOptions()
	machine, allMachines := GetMachineScope()

	sessions, err := getSessions(machine, allMachines)
	if err != nil {
		return err
	}

	sessions = withLogIdle(sessions)
	multiMachine := allMachines || machine != ""

	if opts.Mode == output.ModeJSON {
		return output.RenderJSON(sessions, opts)
	}

	// Build table rows
	var headers []string
	if multiMachine {
		headers = []string{"MACHINE", "NAME", "RUNTIME", "STATUS", "LOG_IDLE", "CWD", "PREVIEW"}
	} else {
		headers = []string{"NAME", "RUNTIME", "STATUS", "LOG_IDLE", "CWD", "PREVIEW"}
	}

	rows := make([][]string, 0, len(sessions))
	for _, s := range sessions {
		status := colorStatus(string(s.Status), opts)
		cwd := truncate(s.CWD, 40)
		preview := truncate(s.Preview, 50)
		logIdle := formatLogIdle(s)
		var row []string
		if multiMachine {
			m := s.Machine
			if m == "" {
				m = "local"
			}
			row = []string{m, s.Name, string(s.Runtime), status, logIdle, cwd, preview}
		} else {
			row = []string{s.Name, string(s.Runtime), status, logIdle, cwd, preview}
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		fmt.Fprintln(os.Stderr, "No agent sessions found.")
		return nil
	}

	if opts.Mode == output.ModePlaintext {
		return output.RenderPlaintext(headers, rows, opts)
	}
	return output.RenderTable(headers, rows, opts)
}
