package handler

import (
	"bytes"

	"github.com/neovim/go-client/nvim"
)

func newBuffer(vim *nvim.Nvim, buffer nvim.Buffer) *Buffer {
	return &Buffer{
		buffer: buffer,
		vim:    vim,
	}
}

type Buffer struct {
	buffer nvim.Buffer
	vim    *nvim.Nvim
}

func (b *Buffer) Write(p []byte) (int, error) {
	lines := bytes.Split(bytes.TrimRight(p, "\r\n"), []byte("\n"))
	for i, line := range lines {
		lines[i] = bytes.TrimRight(line, "\r")
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
