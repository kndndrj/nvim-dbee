package adapters

import (
	"context"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var _ core.Driver = (*redshiftDriver)(nil)

// redshiftDriver is a sql client for redshiftDriver.
// Mainly uses the postgres driver under the hood but with
// custom Layout function to get the table and view names correctly.
type redshiftDriver struct {
	c *builders.Client
}

// Query executes a query and returns the result as an IterResult.
func (c *redshiftDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	con, err := c.c.Conn(ctx)
	if err != nil {
		return nil, err
	}
	cb := func() {
		con.Close()
	}
	defer func() {
		if err != nil {
			cb()
		}
	}()

	rows, err := con.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	rows.SetCallback(cb)
	return rows, nil
}

// Close closes the underlying sql.DB connection.
func (c *redshiftDriver) Close() {
	// TODO: perhaps worth check err return statement here.
	c.c.Close()
}

func (c *redshiftDriver) Columns(opts *core.HelperOptions) ([]*core.Columns, error) {
	query := fmt.Sprintf(`
	SELECT *
	FROM information_schema.columns
	WHERE table_name='%s'
	AND table_schema='%s';
	`, opts.Table, opts.Schema)

	result, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	var column_index int
	var dtype_index int

	for i, header := range result.Header() {
		i := i
		switch header {
		case "column_name":
			column_index = i
		case "data_type":
			dtype_index = i
		}
	}

	var out []*core.Columns

	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("result.Next: %w", err)
		}

		column := &core.Columns{
			Name: row[column_index].(string),
			Type: row[dtype_index].(string),
		}

		out = append(out, column)
	}

	return out, nil
}

// Structure returns the layout of the database. This represents the
// "schema" with all the tables and views. Note that ordering is not
// done here. The ordering is done in the lua frontend.
func (c *redshiftDriver) Structure() ([]*core.Structure, error) {
	query := `
		SELECT
		trim(n.nspname) AS schema_name
		, trim(c.relname) AS table_name
		, CASE
			WHEN c.relkind = 'v' THEN 'VIEW'
			ELSE 'TABLE'
			END AS table_type
			FROM
					pg_class AS c
			INNER JOIN
					pg_namespace AS n ON c.relnamespace = n.oid
			WHERE
					n.nspname NOT IN ('information_schema', 'pg_catalog');
	`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	return getPGStructure(rows)
}
