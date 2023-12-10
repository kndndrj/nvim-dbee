package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// init registers the RedshiftClient to the store,
// i.e. to lua frontend.
func init() {
	_ = register(&Redshift{}, "redshift")
}

var _ core.Adapter = (*Redshift)(nil)

type Redshift struct{}

func (r *Redshift) Connect(rawURL string) (core.Driver, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w: ", err)
	}

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to postgres database: %w", err)
	}

	return &redshiftDriver{
		c: builders.NewClient(db),
	}, nil
}

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
