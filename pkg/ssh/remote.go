package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// RemoteExecutor runs commands on a remote SSH host via the local ssh binary.
type RemoteExecutor struct{}

// Run executes a shell command on the remote host via SSH.
// Returns stdout, stderr, and any error.
func (r *RemoteExecutor) Run(host Host, command string) (stdout, stderr string, err error) {
	return r.RunTimeout(host, command, 30*time.Second)
}

// RunTimeout runs a command with a deadline. When the host requires password
// auth (see PasswordFile), the command is wrapped with sshpass.
func (r *RemoteExecutor) RunTimeout(host Host, command string, timeout time.Duration) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	passwordFile := PasswordFile(host)
	var name string
	var argv []string
	if passwordFile != "" {
		sshpass, lookErr := exec.LookPath("sshpass")
		if lookErr != nil {
			return "", "", fmt.Errorf("host %s needs password auth (sshpass) but sshpass was not found on PATH", host.Name)
		}
		name = sshpass
		argv = append([]string{"-f", passwordFile, "ssh"}, buildSSHArgsAuth(host, command, true)...)
	} else {
		name = "ssh"
		argv = buildSSHArgsAuth(host, command, false)
	}
	cmd := exec.CommandContext(ctx, name, argv...)

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

// PasswordFile returns the sshpass password file to use for host, or "" when
// sshpass should not be used.
//
// Resolution order:
//  1. An explicit "# tagents-sshpass-file <path>" comment in the host block.
//     If the file is missing the path is still returned, so sshpass produces a
//     clear error instead of silently falling back to an unusable key.
//  2. An auto-detected ~/.ssh/<name>-pw when the host sets
//     "PubkeyAuthentication no" and that file exists.
//
// Hosts that use public keys return "" and keep the normal ssh path.
func PasswordFile(host Host) string {
	if host.PasswordFile != "" {
		return host.PasswordFile
	}
	if host.PubkeyDisabled {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		candidate := filepath.Join(home, ".ssh", host.Name+"-pw")
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			return candidate
		}
	}
	return ""
}

// buildSSHArgs builds args for the default key-based path. Retained for
// callers/tests that do not care about auth mode.
func buildSSHArgs(host Host, command string) []string {
	return buildSSHArgsAuth(host, command, false)
}

// buildSSHArgsAuth builds the argument list passed to ssh. In password mode
// BatchMode is omitted because it disables the password prompt sshpass needs,
// and password/keyboard-interactive auth is forced.
func buildSSHArgsAuth(host Host, command string, password bool) []string {
	args := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=5",
	}
	if password {
		args = append(args,
			"-o", "PreferredAuthentications=password,keyboard-interactive",
			"-o", "PubkeyAuthentication=no",
		)
	} else {
		args = append(args, "-o", "BatchMode=yes")
	}
	if host.Port != "" && host.Port != "22" {
		args = append(args, "-p", host.Port)
	}
	if host.User != "" {
		args = append(args, "-l", host.User)
	}
	if host.IdentityFile != "" && !password {
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
