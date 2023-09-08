package output

import (
	"context"

	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/neovim/go-client/nvim"
)

type YankRegisterOutput struct {
	vim       *nvim.Nvim
	formatter Formatter
}

func NewYankRegister(vim *nvim.Nvim, formatter Formatter) *YankRegisterOutput {
	return &YankRegisterOutput{
		vim:       vim,
		formatter: formatter,
	}
}

func (yo *YankRegisterOutput) Write(_ context.Context, result models.Result) error {
	reg := newRegister(yo.vim)
	return yo.formatter.Format(result, reg)
}

type register struct {
	vim *nvim.Nvim
}

func newRegister(vim *nvim.Nvim) *register {
	return &register{
		vim: vim,
	}
}

func (r *register) Write(p []byte) (int, error) {
	err := r.vim.Call("setreg", nil, "", string(p))
	return len(p), err
}
