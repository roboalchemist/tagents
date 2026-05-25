package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/roboalchemist/tagents/pkg/output"
)

var appVersion = "dev"

var (
	flagJSON        bool
	flagPlaintext   bool
	flagNoColor     bool
	flagDebug       bool
	flagVerbose     bool
	flagQuiet       bool
	flagFields      string
	flagJQ          string
	flagMachine     string
	flagAllMachines bool
	flagVersionShort bool
)

var rootCmd = &cobra.Command{
	Use:   "tagents",
	Short: "Manage AI agent fleet across tmux sessions and SSH machines",
	Long: `tagents lets any agent or human manage the AI agent fleet without knowing
tmux or SSH exists. It discovers agent sessions running in tmux across local
and remote machines, reports status, sends messages, reads output, and tails logs.

ENVIRONMENT:
  No API keys required. Uses tmux and SSH from your PATH.
  NO_COLOR           Disable color output

FILES:
  ~/.ssh/config (0600)  SSH machine discovery

EXIT STATUS:
  0  Success
  1  Agent not found or operation failed
  2  Usage error
  3  System error (tmux/ssh unavailable)

BUGS:
  Report bugs to: https://github.com/roboalchemist/tagents/issues`,
	Example: `  tagents list
  tagents status --json
  tagents send my-agent "hello"`,
	Version:       appVersion,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagVersionShort {
			fmt.Printf("tagents version %s\nCopyright (c) 2026 roboalchemist\n", appVersion)
			return nil
		}
		return cmd.Help()
	},
}

func init() {
	rootCmd.SetVersionTemplate("tagents version {{.Version}}\nCopyright (c) 2026 roboalchemist\n")

	// GNU standard: --help output ends with homepage.
	// Long already contains BUGS section (needed for man pages).
	// Help template: Long → Usage → Homepage.
	rootCmd.SetHelpTemplate(`{{with .Long}}{{. | trimRightSpace}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}Homepage: https://github.com/roboalchemist/tagents
`)

	// -V shorthand for --version (capital V; -v is taken by --verbose)
	rootCmd.Flags().BoolVarP(&flagVersionShort, "version-short", "V", false, "Print version and exit")

	pf := rootCmd.PersistentFlags()
	pf.BoolVarP(&flagJSON, "json", "j", false, "JSON output")
	pf.BoolVarP(&flagPlaintext, "plaintext", "p", false, "Tab-separated output for piping")
	pf.BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	pf.BoolVar(&flagDebug, "debug", false, "Verbose logging to stderr")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose output (alias for --debug)")
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-error output")
	pf.BoolVar(&flagQuiet, "silent", false, "Suppress non-error output (alias for --quiet)")
	pf.StringVar(&flagFields, "fields", "", "Comma-separated fields to include in output")
	pf.StringVar(&flagJQ, "jq", "", "JQ expression to filter JSON output")
	pf.StringVar(&flagMachine, "machine", "", "Target a specific SSH host")
	pf.BoolVar(&flagAllMachines, "all-machines", false, "Target all hosts from SSH config")
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		// Emit structured error JSON to stderr when --json is active.
		opts := GetOutputOptions()
		output.RenderError(err.Error(), 1, opts)
	}
	return err
}

// RootCmd returns the root cobra command (used by gendocs).
func RootCmd() *cobra.Command {
	return rootCmd
}

func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = v
}

// DebugLog writes a debug message to stderr when --debug or --verbose is active.
func DebugLog(format string, args ...interface{}) {
	if flagDebug || flagVerbose {
		fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	}
}

// GetMachineScope returns the --machine and --all-machines flag values.
func GetMachineScope() (machine string, allMachines bool) {
	return flagMachine, flagAllMachines
}

// GetOutputOptions builds an output.Options from the current flag state.
func GetOutputOptions() output.Options {
	opts := output.Options{
		NoColor: flagNoColor,
		Debug:   flagDebug || flagVerbose,
		Fields:  flagFields,
		JQ:      flagJQ,
	}
	switch {
	case flagJSON:
		opts.Mode = output.ModeJSON
	case flagPlaintext:
		opts.Mode = output.ModePlaintext
	default:
		opts.Mode = output.ModeTable
	}
	return opts
}
