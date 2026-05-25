package main

import (
	"fmt"
	"os"

	"github.com/roboalchemist/tagents/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	dir := "man/man1"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	header := &doc.GenManHeader{
		Title:   "TAGENTS",
		Section: "1",
	}
	if err := doc.GenManTree(cmd.RootCmd(), header, dir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
