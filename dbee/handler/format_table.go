package handler

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

var _ core.Formatter = (*Table)(nil)

type Table struct{}

func newTable() *Table {
	return &Table{}
}

func (tf *Table) Format(header core.Header, rows []core.Row, opts *core.FormatterOptions) ([]byte, error) {
	tableHeaders := []any{""}
	for _, k := range header {
		tableHeaders = append(tableHeaders, k)
	}
	index := opts.ChunkStart

	var tableRows []table.Row
	for _, row := range rows {
		indexedRow := append([]any{index + 1}, row...)
		tableRows = append(tableRows, table.Row(indexedRow))
		index += 1
	}

	t := table.NewWriter()
	t.AppendHeader(table.Row(tableHeaders))
	t.AppendRows(tableRows)
	t.AppendSeparator()
	t.SetStyle(table.StyleLight)
	t.Style().Format = table.FormatOptions{
		Footer: text.FormatDefault,
		Header: text.FormatDefault,
		Row:    text.FormatDefault,
	}
	t.Style().Options.DrawBorder = false
	t.SuppressTrailingSpaces()
	render := t.Render()

	return []byte(render), nil
}
