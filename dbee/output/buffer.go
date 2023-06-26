package output

import (
	"bufio"
	"bytes"

	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/neovim/go-client/nvim"
)

type BufferOutput struct {
	vim       *nvim.Nvim
	buffer    nvim.Buffer
	formatter Formatter
}

func NewBuffer(vim *nvim.Nvim, formatter Formatter, buffer nvim.Buffer) *BufferOutput {
	return &BufferOutput{
		vim:       vim,
		buffer:    buffer,
		formatter: formatter,
	}
}

func (bo *BufferOutput) SetBuffer(buffer nvim.Buffer) {
	bo.buffer = buffer
}

func (bo *BufferOutput) Write(result models.Result) error {
	_, err := bo.vim.IsBufferValid(bo.buffer)
	if err != nil {
		return err
	}

	buf := newBuf(bo.vim, bo.buffer)

	return bo.formatter.Format(result, buf)
}

func newBuf(vim *nvim.Nvim, buffer nvim.Buffer) *buf {
	return &buf{
		buffer: buffer,
		vim:    vim,
	}
}

type buf struct {
	buffer nvim.Buffer
	vim    *nvim.Nvim
}

func (b *buf) Write(p []byte) (int, error) {
	scanner := bufio.NewScanner(bytes.NewReader(p))
	var lines [][]byte
	for scanner.Scan() {
		lines = append(lines, []byte(scanner.Text()))
	}

	const modifiableOptionName = "modifiable"

	// is the buffer modifiable
	isModifiable := false
	err := b.vim.BufferOption(b.buffer, modifiableOptionName, &isModifiable)
	if err != nil {
		return 0, err
	}

	if !isModifiable {
		err = b.vim.SetBufferOption(b.buffer, modifiableOptionName, true)
		if err != nil {
			return 0, err
		}
	}

	err = b.vim.SetBufferLines(b.buffer, 0, -1, true, lines)
	if err != nil {
		return 0, err
	}

	if !isModifiable {
		err = b.vim.SetBufferOption(b.buffer, modifiableOptionName, false)
		if err != nil {
			return 0, err
		}
	}

	return len(p), nil
}
