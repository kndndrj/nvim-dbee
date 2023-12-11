//go:build cgo && ((darwin && (amd64 || arm64)) || (linux && (amd64 || arm64 || riscv64)))

package adapters

import (
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&Duck{}, "duck", "duckdb")
}

var _ core.Adapter = (*Duck)(nil)

type Duck struct{}

func (d *Duck) Connect(url string) (core.Driver, error) {
	db, err := sql.Open("duckdb", url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to duckdb database: %v", err)
	}

	return &duckDriver{
		c: builders.NewClient(db),
	}, nil
}

func (*Duck) GetHelpers(opts *core.HelperOptions) map[string]string {
	return map[string]string{
		"List":        fmt.Sprintf("SELECT * FROM %q LIMIT 500", opts.Table),
		"Columns":     fmt.Sprintf("DESCRIBE %q", opts.Table),
		"Indexes":     fmt.Sprintf("SELECT * FROM duckdb_indexes() WHERE table_name = '%s'", opts.Table),
		"Constraints": fmt.Sprintf("SELECT * FROM duckdb_constraints() WHERE table_name = '%s'", opts.Table),
	}
}
