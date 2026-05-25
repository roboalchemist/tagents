package cmd

import "github.com/spf13/cobra"

var machinesCmd = &cobra.Command{
	Use:     "machines",
	Short:   "List SSH-config machines with reachability and agent count",
	Long:    `List all machines discovered from ~/.ssh/config, showing reachability and agent count.`,
	Example: `  tagents machines
  tagents machines --json`,
	RunE: runMachines,
}

func init() { rootCmd.AddCommand(machinesCmd) }

func runMachines(cmd *cobra.Command, args []string) error {
	// TODO: implement in TAGENTS-8
	return nil
}
