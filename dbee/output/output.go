package output

import (
	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
)

type Formatter interface {
	Format(result *call.CacheResult, from, to int) ([]byte, error)
	Name() string
}
