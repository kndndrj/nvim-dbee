package adapters

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/vertica/vertica-sql-go"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&Vertica{}, "vertica")
}

var _ core.Adapter = (*Vertica)(nil)

type Vertica struct{}

func (v *Vertica) Connect(url string) (core.Driver, error) {
	db, err := sql.Open("vertica", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to vertica database: %w", err)
	}

	jsonProcessor := func(a any) any {
		b, ok := a.([]byte)
		if !ok {
			return a
		}

		return newPostgresJSONResponse(b)
	}

	return &verticaDriver{
		c: builders.NewClient(db,
			builders.WithCustomTypeProcessor("json", jsonProcessor),
			builders.WithCustomTypeProcessor("jsonb", jsonProcessor),
		),
		url: u,
	}, nil
}

func (*Vertica) GetHelpers(opts *core.TableOptions) map[string]string {
	return map[string]string{
		"List":    fmt.Sprintf("SELECT * FROM %q.%q LIMIT 500", opts.Schema, opts.Table),
		"Columns": fmt.Sprintf("SELECT * FROM v_catalog.columns WHERE table_name='%s' AND table_schema='%s'", opts.Table, opts.Schema),
	}
}
