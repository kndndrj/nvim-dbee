package handler

import (
	"bufio"
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
