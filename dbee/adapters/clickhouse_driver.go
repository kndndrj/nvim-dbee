package adapters

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*clickhouseDriver)(nil)
	_ core.DatabaseSwitcher = (*clickhouseDriver)(nil)
)

type clickhouseDriver struct {
	c    *builders.Client
	opts *clickhouse.Options
}

func (c *clickhouseDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	// run query, fallback to affected rows
	return c.c.QueryUntilNotEmpty(ctx, query, "select changes() as 'Rows Affected'")
}

func (c *clickhouseDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return c.c.ColumnsFromQuery(`
		SELECT name, type
		FROM system.columns
		WHERE
			database='%s' AND
			table='%s'
		`, opts.Schema, opts.Table)
}

func (c *clickhouseDriver) Structure() ([]*core.Structure, error) {
	query := `
        SELECT
            table_schema, table_name, table_type
            FROM information_schema.tables
            WHERE lower(table_schema) != 'information_schema'
        UNION ALL
        SELECT DISTINCT
            lower(table_schema), lower(table_name), table_type
            FROM information_schema.tables
            WHERE lower(table_schema) = 'information_schema'`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	return core.GetGenericStructure(rows, getPGStructureType)
}

func (c *clickhouseDriver) Close() {
	c.c.Close()
}

func (c *clickhouseDriver) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT currentDatabase(), schema_name
        FROM information_schema.schemata
        WHERE schema_name NOT IN (currentDatabase(), 'INFORMATION_SCHEMA')
	`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return "", nil, err
	}

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return "", nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		current = row[0].(string)
		available = append(available, row[1].(string))
	}

	return current, available, nil
}

func (c *clickhouseDriver) SelectDatabase(name string) error {
	c.opts.Auth.Database = name
	c.c.Swap(clickhouse.OpenDB(c.opts))

	return nil
}
