package adapters

import (
	"context"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var _ core.Driver = (*duckDriver)(nil)

type duckDriver struct {
	c *builders.Client
}

func (c *duckDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	return c.c.QueryUntilNotEmpty(ctx, query)
}

func (c *duckDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	query := `
SELECT
	column_name
	, data_type
FROM information_schema.columns
WHERE table_schema = '%s'
	AND table_name = '%s'`
	return c.c.ColumnsFromQuery(query, opts.Schema, opts.Table)
}

func (c *duckDriver) Structure() ([]*core.Structure, error) {
	query := `
SELECT
	s.schema_name
  , t.table_name
  , t.table_type
FROM information_schema.schemata AS s
LEFT JOIN information_schema.tables AS t
	ON s.schema_name = t.table_schema
WHERE s.schema_name NOT IN ('information_schema', 'pg_catalog')
GROUP BY 1, 2, 3;
`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	return getPGStructure(rows)
}

func (c *duckDriver) Close() {
	c.c.Close()
}
