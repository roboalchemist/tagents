package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

type Mode int

const (
	ModeTable     Mode = iota
	ModeJSON
	ModePlaintext
)

type Options struct {
	Mode    Mode
	NoColor bool
	Debug   bool
	Fields  string
	JQ      string
}

func (o Options) ShouldUseColor() bool {
	if o.NoColor {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return true
}

func RenderJSON(v interface{}, opts Options) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func RenderTable(headers []string, rows [][]string, opts Options) error {
	if !opts.ShouldUseColor() {
		color.NoColor = true
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("  ")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
	return nil
}

func RenderPlaintext(headers []string, rows [][]string, opts Options) error {
	fmt.Println(strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Println(strings.Join(row, "\t"))
	}
	return nil
}
