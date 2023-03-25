package output

import (
	"bufio"
	"errors"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/kndndrj/nvim-dbee/clients"
	"github.com/neovim/go-client/nvim"
)

type BufferOutput struct {
	bufnr nvim.Buffer
	vim   *nvim.Nvim
}

func NewBufferOutput(vim *nvim.Nvim, bufnr nvim.Buffer) *BufferOutput {
	return &BufferOutput{
		bufnr: bufnr,
		vim:   vim,
	}
}

func (o *BufferOutput) Set(rows clients.Rows) error {

	header, err := rows.Header()
	if err != nil {
		return err
	}

	if len(header) < 1 {
		return errors.New("no header provided")
	}

	var tableHeaders []any
	for _, k := range header {
		tableHeaders = append(tableHeaders, k)
	}

	// tableRows
	var tableRows []table.Row
	for {
		row, err := rows.Next()
		if row == nil {
			break
		}
		if err != nil {
			return err
		}

		tableRows = append(tableRows, table.Row(row))
	}

	t := table.NewWriter()
	t.AppendHeader(table.Row(tableHeaders))
	t.AppendRows(tableRows)
	t.AppendSeparator()
	t.SetStyle(table.StyleLight)
	t.Style().Format = table.FormatOptions{
    	Footer:    text.FormatDefault,
    	Header:    text.FormatDefault,
    	Row:       text.FormatDefault,
    }
	render := t.Render()

	scanner := bufio.NewScanner(strings.NewReader(render))
	var lines [][]byte
	for scanner.Scan() {
		lines = append(lines, []byte(scanner.Text()))
	}

	err = o.vim.SetBufferLines(o.bufnr, 0, -1, true, lines)
	return err
}
