package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var appVersion = "dev"

var (
	flagJSON        bool
	flagPlaintext   bool
	flagNoColor     bool
	flagDebug       bool
	flagFields      string
	flagJQ          string
	flagMachine     string
	flagAllMachines bool
)

var rootCmd = &cobra.Command{
	Use:   "tagents",
	Short: "Manage AI agent fleet across tmux sessions and SSH machines",
	Long: `tagents lets any agent or human manage the AI agent fleet without knowing
tmux or SSH exists. It discovers agent sessions running in tmux across local
and remote machines, reports status, sends messages, reads output, and tails logs.

ENVIRONMENT:
  No API keys required. Uses tmux and SSH from your PATH.

FILES:
  ~/.ssh/config    SSH machine discovery

EXIT STATUS:
  0  Success
  1  Agent not found or operation failed
  2  Usage error
  3  System error (tmux/ssh unavailable)

BUGS:
  Report bugs to: https://github.com/roboalchemist/tagents/issues`,
	Version:       appVersion,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&flagJSON, "json", "j", false, "JSON output")
	pf.BoolVarP(&flagPlaintext, "plaintext", "p", false, "Tab-separated output for piping")
	pf.BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	pf.BoolVar(&flagDebug, "debug", false, "Verbose logging to stderr")
	pf.StringVar(&flagFields, "fields", "", "Comma-separated fields to include in output")
	pf.StringVar(&flagJQ, "jq", "", "JQ expression to filter JSON output")
	pf.StringVar(&flagMachine, "machine", "", "Target a specific SSH host")
	pf.BoolVar(&flagAllMachines, "all-machines", false, "Target all hosts from SSH config")
}

func Execute() error {
	return rootCmd.Execute()
}

func RootCmd() *cobra.Command {
	return rootCmd
}

func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = v
}

func DebugLog(format string, args ...interface{}) {
	if flagDebug {
		fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	}
}
