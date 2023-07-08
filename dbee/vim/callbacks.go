package vim

import (
	"github.com/neovim/go-client/nvim"
)

type Callbacker struct {
	vim *nvim.Nvim
}

func NewCallbacker(v *nvim.Nvim) *Callbacker {
	return &Callbacker{
		vim: v,
	}
}

// TriggerCallback triggers the callback with id registered in lua
func (cb *Callbacker) TriggerCallback(id string) error {
	return cb.vim.ExecLua(`
		require"dbee.handler.__callbacks".trigger("`+id+`")
	`, nil)
}
