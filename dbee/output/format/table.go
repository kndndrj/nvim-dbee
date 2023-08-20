package format

import (
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/kndndrj/nvim-dbee/dbee/output"
)

var _ output.Formatter = (*Table)(nil)

type Table struct{}

func NewTable() *Table {
	return &Table{}
}

func (cf *Table) Name() string {
	return "table"
}

func (cf *Table) Format(result models.Result, writer io.Writer) error {
	tableHeaders := []any{""}
	for _, k := range result.Header {
		tableHeaders = append(tableHeaders, k)
	}

	index := result.Meta.ChunkStart

	var tableRows []table.Row
	for _, row := range result.Rows {
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

	_, err := writer.Write([]byte(render))
	if err != nil {
		return err
	}
	return nil
}
