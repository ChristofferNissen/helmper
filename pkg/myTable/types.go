package myTable

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

type Table struct {
	writer table.Writer
}

// NewTable creates a new Table with a title and header
func NewTable(title string) *Table {
	t := table.NewWriter()
	t.SetTitle(title)
	t.SetOutputMirror(os.Stdout)
	return &Table{writer: t}
}

// AddRow adds a row to the table
func (t *Table) AddRow(row table.Row) {
	t.writer.AppendRow(row)
}

func (t *Table) AddHeader(header table.Row, configs ...table.RowConfig) {
	t.writer.AppendHeader(header, configs...)
}

// Render renders the table
func (t *Table) Render() {
	t.writer.Render()
}
