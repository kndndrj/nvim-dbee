package output

import (
	"bufio"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/kndndrj/nvim-dbee/conn"
	"github.com/neovim/go-client/nvim"
)

type Ui interface {
	Open() (nvim.Window, nvim.Buffer, error)
	Close() error
}

type BufferOutput struct {
	vim           *nvim.Nvim
	window        nvim.Window
	buffer        nvim.Buffer
	windowCommand string
}

func NewBufferOutput() *BufferOutput {
	return &BufferOutput{
		windowCommand: "bo 15split",
		buffer:        -1,
		window:        -1,
	}
}
func (bo *BufferOutput) SetConfig(windowCommand string, vim *nvim.Nvim) {
	bo.windowCommand = windowCommand
	bo.vim = vim
}

func (bo *BufferOutput) Open() error {
	// buffer
	bufValid, _ := bo.vim.IsBufferValid(bo.buffer)
	if !bufValid {
		buf, err := bo.vim.CreateBuffer(false, true)
		if err != nil {
			return err
		}
		bo.buffer = buf
	}

	// window
	winValid, _ := bo.vim.IsWindowValid(bo.window)
	if !winValid {
		err := bo.vim.Command(bo.windowCommand)
		if err != nil {
			return err
		}
		win, err := bo.vim.CurrentWindow()
		if err != nil {
			return err
		}
		bo.window = win
	}

	return bo.vim.SetBufferToWindow(bo.window, bo.buffer)

}

// TODO:
func (bo *BufferOutput) Close() error {
	return nil
}

func (bo *BufferOutput) Write(result conn.Result) error {

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

	err := bo.Open()
	if err != nil {
		return err
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
