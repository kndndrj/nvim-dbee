package output

import (
	"bufio"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/neovim/go-client/nvim"
)

type BufferOutput struct {
	vim    *nvim.Nvim
	buffer nvim.Buffer
}

func NewBufferOutput(vim *nvim.Nvim) *BufferOutput {

	return &BufferOutput{
		vim:    vim,
		buffer: -1,
	}
}
func (bo *BufferOutput) SetBuffer(buffer nvim.Buffer) {
	bo.buffer = buffer
}

func (bo *BufferOutput) Write(result conn.Result) error {
	_, err := bo.vim.IsBufferValid(bo.buffer)
	if err != nil {
		return err
	}

	var tableHeaders []any
	for _, k := range result.Header {
		tableHeaders = append(tableHeaders, k)
	}

	var tableRows []table.Row
	for _, row := range result.Rows {
		tableRows = append(tableRows, table.Row(row))
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
	render := t.Render()

	scanner := bufio.NewScanner(strings.NewReader(render))
	var lines [][]byte
	for scanner.Scan() {
		lines = append(lines, []byte(scanner.Text()))
	}

	err = bo.vim.SetBufferOption(bo.buffer, "modifiable", true)
	if err != nil {
		return err
	}
	err = bo.vim.SetBufferLines(bo.buffer, 0, -1, true, lines)
	if err != nil {
		return err
	}
	return bo.vim.SetBufferOption(bo.buffer, "modifiable", false)
}
