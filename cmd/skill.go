package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var skillMD string
var commandsRef string
var skillFS embed.FS

func SetSkillData(md, commands string, fsys embed.FS) {
	skillMD = md
	commandsRef = commands
	skillFS = fsys
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage Claude Code skill installation",
}

var skillPrintCmd = &cobra.Command{
	Use:   "print",
	Short: "Print SKILL.md to stdout",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(skillMD)
	},
}

var skillAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Install skill to ~/.claude/skills/tagents/",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dest := filepath.Join(home, ".claude", "skills", "tagents")
		if err := os.MkdirAll(dest, 0755); err != nil {
			return err
		}
		return fs.WalkDir(skillFS, "skill", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel("skill", path)
			target := filepath.Join(dest, rel)
			if d.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			data, err := skillFS.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, 0644)
		})
	},
}

func init() {
	skillCmd.AddCommand(skillPrintCmd)
	skillCmd.AddCommand(skillAddCmd)
	rootCmd.AddCommand(skillCmd)
}
