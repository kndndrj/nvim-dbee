package format

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var _ call.Formatter = (*Table)(nil)

type Table struct{}

func NewTable() *Table {
	return &Table{}
}

func (tf *Table) Format(header models.Header, rows []models.Row, meta *models.Meta) ([]byte, error) {
	tableHeaders := []any{""}
	for _, k := range header {
		tableHeaders = append(tableHeaders, k)
	}
	index := meta.ChunkStart

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
	render := t.Render()

	return []byte(render), nil
}
