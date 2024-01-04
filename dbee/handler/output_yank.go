package handler

import (
	"fmt"

	"github.com/neovim/go-client/nvim"
)

type YankRegister struct {
	vim      *nvim.Nvim
	register string
}

func newYankRegister(vim *nvim.Nvim, register string) *YankRegister {
	return &YankRegister{
		vim:      vim,
		register: register,
	}
}

func (yr *YankRegister) Write(p []byte) (int, error) {
	err := yr.vim.Call("setreg", nil, yr.register, string(p))
	if err != nil {
		return 0, fmt.Errorf("r.vim.Call: %w", err)
	}

	return len(p), err
}
