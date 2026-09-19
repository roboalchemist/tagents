package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/roboalchemist/tagents/pkg/launch"
	"github.com/roboalchemist/tagents/pkg/output"
	"github.com/roboalchemist/tagents/pkg/tmux"
	"github.com/spf13/cobra"
)

var (
	flagCreateRuntime  string
	flagCreateModel    string
	flagCreateCommand  string
	flagCreateCwd      string
	flagCreatePrompt   string
	flagCreateRepo     string
	flagCreateWorktree string
	flagCreateBranch   string
	flagCreateFrom     string
	flagCreateWait     time.Duration
	flagCreateForce    bool
	flagCreateDryRun   bool
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new agent session",
	Long: `Create a new tmux session and launch an agent runtime in it.

By default it launches OpenCode in the current directory. Use --runtime and
--model to pick a different runtime or model, --cwd to choose the working
directory, or --repo/--worktree/--branch to create a git worktree first. Pass
--prompt to send an initial message once the agent is running.

create targets the local machine only. Follow it with tagents send/read/wait to
guide and inspect the new agent.

Examples:
  tagents create my-agent
  tagents create worker --runtime opencode --model 'haiku[1m]' --cwd ~/work
  tagents create PROJ-123 --repo ~/src/example-repo \
    --worktree ~/worktrees/PROJ-123 --branch dev/proj-123 \
    --prompt "/v/one-shot PROJ-123"`,
	Example: `  tagents create my-agent
  tagents create worker --runtime opencode --model 'haiku[1m]'
  tagents create PROJ-123 --repo ~/src/example-repo \
    --worktree ~/worktrees/PROJ-123 --branch dev/proj-123 \
    --prompt "/v/one-shot PROJ-123"`,
	SuggestFor: []string{"new", "spawn", "start"},
	Args:       cobra.ExactArgs(1),
	RunE:       runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVar(&flagCreateRuntime, "runtime", "opencode", "Agent runtime: opencode, claude, codex, or pi")
	f.StringVar(&flagCreateModel, "model", "", "Model to pass to the runtime (e.g. 'haiku[1m]')")
	f.StringVar(&flagCreateCommand, "command", "", "Explicit launch command (overrides --runtime/--model)")
	f.StringVar(&flagCreateCwd, "cwd", "", "Working directory for the session (default: current directory)")
	f.StringVar(&flagCreatePrompt, "prompt", "", "Initial message to send once the agent is running")
	f.StringVar(&flagCreateRepo, "repo", "", "Git repo to create the worktree from")
	f.StringVar(&flagCreateWorktree, "worktree", "", "Worktree directory to create")
	f.StringVar(&flagCreateBranch, "branch", "", "Branch to create for the worktree")
	f.StringVar(&flagCreateFrom, "from", "origin/master", "Start point for the new worktree branch")
	f.DurationVar(&flagCreateWait, "wait", 20*time.Second, "Max time to wait for the agent to start")
	f.BoolVar(&flagCreateForce, "force", false, "Replace an existing session or worktree of the same name")
	f.BoolVar(&flagCreateDryRun, "dry-run", false, "Print the plan without changing anything")
	rootCmd.AddCommand(createCmd)
}

type createResult struct {
	Name     string `json:"name"`
	Runtime  string `json:"runtime,omitempty"`
	Model    string `json:"model,omitempty"`
	CWD      string `json:"cwd"`
	Command  string `json:"command"`
	Worktree string `json:"worktree,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Prompt   string `json:"prompt,omitempty"`
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("session name must not be empty")
	}

	if machine, all := GetMachineScope(); machine != "" || all {
		return fmt.Errorf("create currently supports the local machine only")
	}

	launchCmd, err := launch.Spec{
		Runtime: flagCreateRuntime,
		Model:   flagCreateModel,
		Command: flagCreateCommand,
	}.CommandLine()
	if err != nil {
		return err
	}

	cwd, err := resolveCreateCwd()
	if err != nil {
		return err
	}

	res := createResult{
		Name:    name,
		Command: launchCmd,
		CWD:     cwd,
		Prompt:  flagCreatePrompt,
	}
	if flagCreateCommand == "" {
		rt, _ := launch.NormalizeRuntime(flagCreateRuntime)
		res.Runtime = rt
		res.Model = flagCreateModel
	}
	if flagCreateWorktree != "" {
		res.Worktree = expandTilde(flagCreateWorktree)
		res.Branch = flagCreateBranch
	}

	if flagCreateDryRun {
		return renderCreateResult(res, true)
	}

	if flagCreateWorktree != "" {
		if err := prepareWorktree(); err != nil {
			return err
		}
	}

	client := tmux.NewLocalClient()
	if client.SessionExists(name) {
		if !flagCreateForce {
			return fmt.Errorf("session %q already exists (use --force to replace)", name)
		}
		if err := client.KillSession(name); err != nil {
			return err
		}
	}

	if err := client.NewSession(name, cwd); err != nil {
		return err
	}

	if err := client.SendKeys(name, launchCmd); err != nil {
		_ = client.KillSession(name)
		return fmt.Errorf("launching agent in %q: %w", name, err)
	}

	rt := ""
	if flagCreateCommand == "" {
		rt, _ = launch.NormalizeRuntime(flagCreateRuntime)
	}
	if err := waitForAgent(client, name, rt, flagCreateWait); err != nil {
		// The session exists; report but do not tear it down.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	if flagCreatePrompt != "" {
		if err := client.SendKeys(name, flagCreatePrompt); err != nil {
			return fmt.Errorf("sending initial prompt to %q: %w", name, err)
		}
	}

	return renderCreateResult(res, false)
}

// resolveCreateCwd determines the session working directory: the worktree when
// one is being created, otherwise --cwd, otherwise the current directory.
func resolveCreateCwd() (string, error) {
	switch {
	case flagCreateWorktree != "":
		return expandTilde(flagCreateWorktree), nil
	case flagCreateCwd != "":
		return expandTilde(flagCreateCwd), nil
	default:
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("determining current directory: %w", err)
		}
		return wd, nil
	}
}

// prepareWorktree creates the git worktree requested by --repo/--worktree/--branch.
// With --force an existing worktree at that path is removed first.
func prepareWorktree() error {
	repo := expandTilde(flagCreateRepo)
	worktree := expandTilde(flagCreateWorktree)
	if repo == "" || worktree == "" {
		return fmt.Errorf("--repo and --worktree must be used together")
	}
	if fi, err := os.Stat(repo); err != nil || !fi.IsDir() {
		return fmt.Errorf("repo %q is not a directory", repo)
	}
	if _, err := os.Stat(worktree); err == nil {
		if !flagCreateForce {
			return fmt.Errorf("worktree %q already exists (use --force to replace)", worktree)
		}
		if out, err := exec.Command("git", "-C", repo, "worktree", "remove", "--force", worktree).CombinedOutput(); err != nil {
			return fmt.Errorf("removing existing worktree %q: %v: %s", worktree, err, strings.TrimSpace(string(out)))
		}
	}

	addArgs := []string{"-C", repo, "worktree", "add", worktree}
	if flagCreateBranch != "" {
		addArgs = append(addArgs, "-b", flagCreateBranch)
	}
	if flagCreateFrom != "" {
		addArgs = append(addArgs, flagCreateFrom)
	}
	if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// waitForAgent waits until the pane's foreground process is no longer a shell
// and, for runtimes with a known TUI marker, until that marker is visible.
// Sending input before the TUI is ready causes the first message to be dropped,
// so callers send --prompt only after this returns.
func waitForAgent(client *tmux.Client, name, rt string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	started := false
	for time.Now().Before(deadline) {
		running, err := client.IsPaneRunning(name)
		if err == nil && running {
			started = true
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if !started {
		return fmt.Errorf("agent %q did not start within %s", name, timeout)
	}

	markers := readyMarkers(rt)
	if len(markers) == 0 {
		// No reliable marker for this runtime — allow the TUI to settle.
		settle := 6 * time.Second
		if remaining := time.Until(deadline); remaining < settle {
			settle = remaining
		}
		if settle > 0 {
			time.Sleep(settle)
		}
		return nil
	}

	for time.Now().Before(deadline) {
		content, err := client.CapturePane(name, 50)
		if err == nil {
			lower := strings.ToLower(content)
			for _, m := range markers {
				if strings.Contains(lower, m) {
					return nil
				}
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
	return fmt.Errorf("agent %q did not become ready within %s", name, timeout)
}

// readyMarkers returns pane-content substrings (lowercased) that indicate the
// runtime's interactive input is accepting keystrokes. An empty result means the
// runtime has no reliable marker and callers should fall back to a fixed settle.
func readyMarkers(rt string) []string {
	switch rt {
	case launch.OpenCode:
		return []string{"ctrl+p commands", "ask anything"}
	case launch.Pi:
		return []string{"show full startup help", "look up its docs"}
	default:
		return nil
	}
}

func renderCreateResult(res createResult, dryRun bool) error {
	opts := GetOutputOptions()

	if opts.Mode == output.ModeJSON {
		return output.RenderJSON(res, opts)
	}

	prefix := "Created"
	if dryRun {
		prefix = "[dry-run] Would create"
	}
	fmt.Printf("%s session %s\n", prefix, res.Name)
	fmt.Printf("  runtime: %s\n", res.Runtime)
	if res.Model != "" {
		fmt.Printf("  model:   %s\n", res.Model)
	}
	fmt.Printf("  cwd:     %s\n", res.CWD)
	fmt.Printf("  command: %s\n", res.Command)
	if res.Worktree != "" {
		fmt.Printf("  worktree: %s (%s)\n", res.Worktree, res.Branch)
	}
	if res.Prompt != "" {
		verb := "sent prompt"
		if dryRun {
			verb = "would send prompt"
		}
		fmt.Printf("  %s: %s\n", verb, res.Prompt)
	}
	return nil
}

// expandTilde expands a leading ~ or ~/ to the user's home directory. tmux -c
// and git do not expand ~ themselves.
func expandTilde(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
