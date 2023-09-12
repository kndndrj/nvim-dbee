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

func (cf *Table) Format(result models.IterResult, writer io.Writer) error {
	tableHeaders := []any{""}
	header, err := result.Header()
	if err != nil {
		return err
	}
	for _, k := range header {
		tableHeaders = append(tableHeaders, k)
	}
	meta, err := result.Meta()
	if err != nil {
		return err
	}

	index := meta.ChunkStart

	var tableRows []table.Row
	for {
		row, err := result.Next()
		if err != nil {
			return err
		}
		if row == nil {
			break
		}

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

	_, err = writer.Write([]byte(render))
	if err != nil {
		return err
	}
	return nil
}
