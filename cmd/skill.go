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
var skillFS embed.FS

func SetSkillData(md string, fsys embed.FS) {
	skillMD = md
	skillFS = fsys
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage Claude Code skill installation",
	Long: `Manage the Claude Code skill bundled with tagents.

The skill teaches Claude Code how to use tagents to manage agent fleets.`,
	Example: `  tagents skill print
  tagents skill add`,
}

var skillPrintCmd = &cobra.Command{
	Use:   "print",
	Short: "Print SKILL.md to stdout",
	Example: `  tagents skill print
  tagents skill print | less`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(skillMD)
		return nil
	},
}

var skillAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Install skill to ~/.claude/skills/tagents/",
	Long: `Install the bundled tagents Claude Code skill to ~/.claude/skills/tagents/.

After installation, tagents will appear in Claude Code's skill list and agents
can use it to manage the fleet.`,
	Example: `  tagents skill add`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dest := filepath.Join(home, ".claude", "skills", "tagents")
		if err := os.MkdirAll(dest, 0755); err != nil {
			return err
		}
		if err := fs.WalkDir(skillFS, "skill", func(path string, d fs.DirEntry, err error) error {
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
		}); err != nil {
			return err
		}

		fmt.Printf("Skill installed to %s\n", dest)

		// Verify SKILL.md was installed
		if _, err := os.Stat(filepath.Join(dest, "SKILL.md")); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: SKILL.md not found in %s after installation\n", dest)
		}

		return nil
	},
}

func init() {
	skillCmd.AddCommand(skillPrintCmd)
	skillCmd.AddCommand(skillAddCmd)
	rootCmd.AddCommand(skillCmd)
}
