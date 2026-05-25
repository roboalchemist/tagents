package cmd

import (
	"fmt"
	"sort"

	"github.com/roboalchemist/tagents/pkg/output"
	"github.com/roboalchemist/tagents/pkg/runtime"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Fleet status summary",
	Long: `Show fleet summary counts: total, busy, idle, dead.

Counts are per-machine when multi-machine scope is used.

Examples:
  tagents status
  tagents status --json
  tagents status --all-machines`,
	Example: `  tagents status
  tagents status --json
  tagents status --all-machines --json`,
	RunE: runStatus,
}

func init() { rootCmd.AddCommand(statusCmd) }

type machineStatus struct {
	Total int `json:"total"`
	Busy  int `json:"busy"`
	Idle  int `json:"idle"`
	Dead  int `json:"dead"`
}

type fleetStatus struct {
	Overall  machineStatus            `json:"overall"`
	Machines map[string]machineStatus `json:"machines,omitempty"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	opts := GetOutputOptions()
	machine, allMachines := GetMachineScope()

	sessions, err := getSessions(machine, allMachines)
	if err != nil {
		return err
	}

	byMachine := make(map[string]*machineStatus)
	overall := &machineStatus{}

	for _, s := range sessions {
		m := s.Machine
		if m == "" {
			m = "local"
		}
		if byMachine[m] == nil {
			byMachine[m] = &machineStatus{}
		}
		ms := byMachine[m]
		ms.Total++
		overall.Total++
		switch s.Status {
		case runtime.Busy:
			ms.Busy++
			overall.Busy++
		case runtime.Idle:
			ms.Idle++
			overall.Idle++
		case runtime.Dead:
			ms.Dead++
			overall.Dead++
		}
	}

	fleet := fleetStatus{Overall: *overall}
	if allMachines || machine != "" {
		fleet.Machines = make(map[string]machineStatus)
		for k, v := range byMachine {
			fleet.Machines[k] = *v
		}
	}

	if opts.Mode == output.ModeJSON {
		return output.RenderJSON(fleet, opts)
	}

	// Collect machine names in sorted order for deterministic output.
	machineNames := make([]string, 0, len(byMachine))
	for name := range byMachine {
		machineNames = append(machineNames, name)
	}
	sort.Strings(machineNames)

	headers := []string{"MACHINE", "TOTAL", "BUSY", "IDLE", "DEAD"}
	var rows [][]string
	for _, name := range machineNames {
		ms := byMachine[name]
		rows = append(rows, []string{
			name,
			fmt.Sprint(ms.Total),
			fmt.Sprint(ms.Busy),
			fmt.Sprint(ms.Idle),
			fmt.Sprint(ms.Dead),
		})
	}

	if opts.Mode == output.ModePlaintext {
		return output.RenderPlaintext(headers, rows, opts)
	}
	return output.RenderTable(headers, rows, opts)
}
