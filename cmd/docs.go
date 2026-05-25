package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var readmeContents string

func SetReadmeContents(content string) { readmeContents = content }

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Display full documentation",
	Long:  "Display the complete documentation from README.md.\n\nExamples:\n  tagents docs\n  tagents docs | less",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(readmeContents)
	},
}

func init() { rootCmd.AddCommand(docsCmd) }
