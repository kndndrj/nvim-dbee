package output

import (
	"io"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

type Formatter interface {
	Format(result models.IterResult, writer io.Writer) error
	Name() string
}
