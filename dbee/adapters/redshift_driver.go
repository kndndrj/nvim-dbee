package adapters

import (
	"context"
	"errors"
	"fmt"

	_ "github.com/lib/pq"

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

func (c *redshiftDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	query := fmt.Sprintf(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name='%s'
		AND table_schema='%s';
	`, opts.Table, opts.Schema)

	result, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	var out []*core.Column

	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("result.Next: %w", err)
		}

		if len(row) < 2 {
			return nil, errors.New("could not retrieve column info: insufficient data")
		}

		name, ok := row[0].(string)
		if !ok {
			return nil, errors.New("could not retrieve column info: name not a string")
		}

		typ, ok := row[1].(string)
		if !ok {
			return nil, errors.New("could not retrieve column info: type not a string")
		}

		column := &core.Column{
			Name: name,
			Type: typ,
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
