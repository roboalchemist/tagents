package main

import (
	"embed"
	"os"
	"strings"

	"github.com/roboalchemist/tagents/cmd"
)

var version = "dev"

//go:embed README.md
var readmeContents string

//go:embed skill/SKILL.md
var skillMD string

//go:embed skill
var skillFS embed.FS

// exitCode categorizes an error into a GNU-standard exit code.
//
//	0  success
//	1  user/runtime error (agent not found, tmux error, etc.)
//	2  usage error (unknown flag, wrong args, required flag missing)
//	3  system error (tmux/ssh not installed)
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	msg := err.Error()
	// Cobra parse/usage errors → exit 2
	if strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "unknown command") ||
		strings.Contains(msg, "accepts") ||
		strings.Contains(msg, "required flag") {
		return 2
	}
	// System-level errors → exit 3
	if strings.Contains(msg, "executable file not found") ||
		strings.Contains(msg, "tmux: command not found") ||
		strings.Contains(msg, "no such file or directory") {
		return 3
	}
	return 1
}

func main() {
	cmd.SetVersion(version)
	cmd.SetReadmeContents(readmeContents)
	cmd.SetSkillData(skillMD, skillFS)
	if err := cmd.Execute(); err != nil {
		os.Exit(exitCode(err))
	}
}
