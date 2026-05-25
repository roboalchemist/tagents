package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// RemoteExecutor runs commands on a remote SSH host via the local ssh binary.
type RemoteExecutor struct{}

// Run executes a shell command on the remote host via SSH.
// Returns stdout, stderr, and any error.
func (r *RemoteExecutor) Run(host Host, command string) (stdout, stderr string, err error) {
	return r.RunTimeout(host, command, 30*time.Second)
}

// RunTimeout runs a command with a deadline.
func (r *RemoteExecutor) RunTimeout(host Host, command string, timeout time.Duration) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := buildSSHArgs(host, command)
	cmd := exec.CommandContext(ctx, "ssh", args...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if runErr := cmd.Run(); runErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", "", fmt.Errorf("SSH timeout connecting to %s", host.Name)
		}
		return outBuf.String(), errBuf.String(), fmt.Errorf("ssh %s: %w", host.Name, runErr)
	}
	return outBuf.String(), errBuf.String(), nil
}

// Ping checks if a host is reachable via SSH within the given timeout.
func (r *RemoteExecutor) Ping(host Host, timeout time.Duration) bool {
	_, _, err := r.RunTimeout(host, "true", timeout)
	return err == nil
}

func buildSSHArgs(host Host, command string) []string {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=5",
		"-o", "BatchMode=yes",
	}
	if host.Port != "" && host.Port != "22" {
		args = append(args, "-p", host.Port)
	}
	if host.User != "" {
		args = append(args, "-l", host.User)
	}
	if host.IdentityFile != "" {
		args = append(args, "-i", host.IdentityFile)
	}
	target := host.Name
	if host.HostName != "" {
		target = host.HostName
	}
	args = append(args, target)
	args = append(args, command)
	return args
}
