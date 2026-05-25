package tmux

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Executor runs tmux commands. Interface allows mocking in tests.
type Executor interface {
	Run(args ...string) (string, error)
}

// OSExecutor runs real tmux commands via os/exec.
type OSExecutor struct{}

func (e *OSExecutor) Run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}
