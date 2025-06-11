package adapters

import (
	"context"
	"fmt"

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
			database AS table_schema,
			name AS table_name,
			CASE
				WHEN engine IN (
					'MergeTree', 'ReplacingMergeTree', 'SummingMergeTree', 'AggregatingMergeTree',
					'CollapsingMergeTree', 'VersionedCollapsingMergeTree', 'GraphiteMergeTree',
					'TinyLog', 'Log', 'StripeLog', 'Memory', 'Buffer', 'Distributed'
				) THEN 'BASE TABLE'
				WHEN engine = 'View' THEN 'VIEW'
				WHEN engine = 'MaterializedView' THEN 'VIEW'
				WHEN engine = 'LiveView' THEN 'VIEW'
				WHEN database IN ('system', 'information_schema') THEN 'SYSTEM TABLE'
				ELSE 'UNKNOWN'
			END AS table_type
		FROM system.tables
		ORDER BY database, name
		`

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
		SELECT
    		currentDatabase() AS current_db,
    		name AS schema_name
		FROM system.databases
		WHERE name NOT IN (currentDatabase(), 'INFORMATION_SCHEMA')
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
	oldDB := c.opts.Auth.Database
	c.opts.Auth.Database = name

	db := clickhouse.OpenDB(c.opts)
	if err := db.PingContext(context.Background()); err != nil {
		c.opts.Auth.Database = oldDB
		return fmt.Errorf("pinging connection failed with %v", err)
	}

	c.c.Swap(db)
	return nil
}
