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
	Long:  `Display the complete documentation from README.md.`,
	Example: `  tagents docs
  tagents docs | less`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(readmeContents)
		return nil
	},
}

func init() { rootCmd.AddCommand(docsCmd) }
