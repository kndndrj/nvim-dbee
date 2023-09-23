package handler

import "github.com/neovim/go-client/nvim"

type YankRegister struct {
	vim *nvim.Nvim
}

func newYankRegister(vim *nvim.Nvim) *YankRegister {
	return &YankRegister{
		vim: vim,
	}
}

func (r *YankRegister) Write(p []byte) (int, error) {
	err := r.vim.Call("setreg", nil, "", string(p))
	return len(p), err
}
