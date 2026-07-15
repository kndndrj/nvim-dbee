package handler

import (
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

var _ core.Formatter = (*Table)(nil)

type Table struct {
	maxColumnWidth int
}

func newTable(maxColumnWidth int) *Table {
	return &Table{maxColumnWidth: maxColumnWidth}
}

func clipCell(col string, maxLen int) string {
	return text.Trim(strings.Join(strings.Fields(col), " "), maxLen)
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

	if tf.maxColumnWidth > 0 {
		columnConfigs := make([]table.ColumnConfig, 0, len(tableHeaders))
		for i := range tableHeaders {
			columnConfigs = append(columnConfigs, table.ColumnConfig{
				Number:           i + 1,
				WidthMax:         tf.maxColumnWidth,
				WidthMaxEnforcer: clipCell,
			})
		}
		t.SetColumnConfigs(columnConfigs)
	}

	render := t.Render()

	return []byte(render), nil
}
