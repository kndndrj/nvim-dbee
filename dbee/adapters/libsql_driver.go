package adapters

import (
	"context"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var _ core.Driver = (*libSQLDriver)(nil)

type libSQLDriver struct {
	c *builders.Client
}

func (c *libSQLDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	// run query, fallback to affected rows
	return c.c.QueryUntilNotEmpty(ctx, query, "select changes() as 'Rows Affected'")
}

func (c *libSQLDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return c.c.ColumnsFromQuery("SELECT name, type FROM pragma_table_info('%s')", opts.Table)
}

func (c *libSQLDriver) Structure() ([]*core.Structure, error) {
	query := `SELECT name FROM sqlite_schema WHERE type ='table'`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	var schema []*core.Structure
	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}

		// We know for a fact there is only one string field (see query above)
		table := row[0].(string)
		schema = append(schema, &core.Structure{
			Name:   table,
			Schema: "",
			Type:   core.StructureTypeTable,
		})
	}

	return schema, nil
}

func (c *libSQLDriver) Close() {
	c.c.Close()
}
