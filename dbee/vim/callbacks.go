package vim

import (
	"fmt"

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
func (cb *Callbacker) TriggerCallback(id string, success bool) error {
	return cb.vim.ExecLua(fmt.Sprintf(`require("dbee.handler.__callbacks").trigger("%s", %t)`, id, success), nil)
}
