package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/lib/pq"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*redshiftDriver)(nil)
	_ core.DatabaseSwitcher = (*redshiftDriver)(nil)
)

// redshiftDriver is a sql client for redshiftDriver.
// Mainly uses the postgres driver under the hood but with
// custom Layout function to get the table and view names correctly.
type redshiftDriver struct {
	c             *builders.Client
	connectionURL *url.URL
}

// Query executes a query and returns the result as an IterResult.
func (r *redshiftDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	return r.c.QueryUntilNotEmpty(ctx, query)
}

// Close closes the underlying sql.DB connection.
func (r *redshiftDriver) Close() {
	r.c.Close()
}

func (r *redshiftDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return r.c.ColumnsFromQuery(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE
			table_schema='%s' AND
			table_name='%s'
		`, opts.Schema, opts.Table)
}

// Structure returns the layout of the database. This represents the
// "schema" with all the tables and views. Note that ordering is not
// done here. The ordering is done in the lua frontend.
func (r *redshiftDriver) Structure() ([]*core.Structure, error) {
	query := `
		SELECT
			trim(n.nspname) AS schema_name,
			trim(c.relname) AS table_name,
			CASE
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

	rows, err := r.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return core.GetGenericStructure(rows, getPGStructureType)
}

func (r *redshiftDriver) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT current_database() AS current, datname
		FROM pg_database
		WHERE datistemplate = false
		  AND datname != current_database();`

	rows, err := r.Query(context.Background(), query)
	if err != nil {
		return "", nil, err
	}

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return "", nil, err
		}

		// current database is the first column, available databases are the rest
		current = row[0].(string)
		available = append(available, row[1].(string))
	}

	return current, available, nil
}

func (r *redshiftDriver) SelectDatabase(name string) error {
	r.connectionURL.Path = fmt.Sprintf("/%s", name)
	db, err := sql.Open("postgres", r.connectionURL.String())
	if err != nil {
		return fmt.Errorf("unable to switch databases: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("unable to ping redshift: %w", err)
	}

	r.c.Swap(db)
	return nil
}
