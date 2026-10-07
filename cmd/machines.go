package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/roboalchemist/tagents/pkg/output"
	"github.com/roboalchemist/tagents/pkg/session"
	"github.com/roboalchemist/tagents/pkg/ssh"
	"github.com/spf13/cobra"
)

var machinesCmd = &cobra.Command{
	Use:   "machines",
	Short: "List available SSH machines",
	Long: `List all machines available from ~/.ssh/config.

Shows name, reachability, agent count, and auth method. Pings each host in parallel.
Auth is "key" for public-key hosts and "sshpass" for hosts that only accept a
password (see the "# tagents-sshpass-file" directive and PubkeyAuthentication).

Examples:
  tagents machines
  tagents machines --json`,
	Example: `  tagents machines
  tagents machines --json | jq '.[] | select(.reachable)'`,
	SuggestFor: []string{"hosts", "nodes"},
	RunE:       runMachines,
}

func init() { rootCmd.AddCommand(machinesCmd) }

type machineInfo struct {
	Name      string `json:"name"`
	Reachable bool   `json:"reachable"`
	Agents    int    `json:"agents"`
	Auth      string `json:"auth"`
}

// hostAuth reports how tagents authenticates to a host: "sshpass" when a
// password file is configured/detected, otherwise "key".
func hostAuth(h ssh.Host) string {
	if ssh.PasswordFile(h) != "" {
		return "sshpass"
	}
	return "key"
}

func runMachines(cmd *cobra.Command, args []string) error {
	opts := GetOutputOptions()

	sshHosts, err := ssh.ParseSSHConfig("")
	if err != nil {
		return fmt.Errorf("reading SSH config: %w", err)
	}

	type result struct {
		host      ssh.Host
		reachable bool
		agents    int
	}

	results := make([]result, len(sshHosts))
	var wg sync.WaitGroup
	for i, h := range sshHosts {
		i, h := i, h
		wg.Add(1)
		go func() {
			defer wg.Done()
			exec := &ssh.RemoteExecutor{}
			reachable := exec.Ping(h, 5*time.Second)
			agentCount := 0
			if reachable {
				sessions, err := session.DiscoverRemote(h, 5*time.Second)
				if err == nil {
					agentCount = len(sessions)
				}
			}
			results[i] = result{host: h, reachable: reachable, agents: agentCount}
		}()
	}
	wg.Wait()

	var machines []machineInfo
	for _, r := range results {
		machines = append(machines, machineInfo{
			Name:      r.host.Name,
			Reachable: r.reachable,
			Agents:    r.agents,
			Auth:      hostAuth(r.host),
		})
	}

	if opts.Mode == output.ModeJSON {
		return output.RenderJSON(machines, opts)
	}

	headers := []string{"NAME", "REACHABLE", "AGENTS", "AUTH"}
	rows := make([][]string, 0, len(machines))
	for _, m := range machines {
		reachable := "no"
		if m.Reachable {
			reachable = "yes"
		}
		auth := m.Auth
		if auth == "" {
			auth = "key"
		}
		rows = append(rows, []string{m.Name, reachable, fmt.Sprintf("%d", m.Agents), auth})
	}

	if len(rows) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "No machines found in SSH config.")
		return nil
	}

	if opts.Mode == output.ModePlaintext {
		return output.RenderPlaintext(headers, rows, opts)
	}
	return output.RenderTable(headers, rows, opts)
}
