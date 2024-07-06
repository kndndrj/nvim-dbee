package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/snowflakedb/gosnowflake"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*snowflakeDriver)(nil)
	_ core.DatabaseSwitcher = (*snowflakeDriver)(nil)
)

// Custom URL type for snowflake.
// gosnowflake does not support the snowflake:// scheme
// , it expects the full connection string excluding the scheme.
// e.g. "snowflake://user:password@account/db" -> "user:password@account/db"
type SnowflakeURL struct {
	url.URL
}

func (c *SnowflakeURL) String() string {
	result := ""
	if c.User != nil {
		result += c.User.String() + "@"
	}
	result += c.Host
	result += c.Path
	if c.RawQuery != "" {
		result += "?" + c.RawQuery
	}
	if c.Fragment != "" {
		result += "#" + c.Fragment
	}
	return result
}

// snowflakeDriver is a sql client for snowflakeDriver.
type snowflakeDriver struct {
	c             *builders.Client
	connectionURL *SnowflakeURL
}

// Query executes a query and returns the result as an IterResult.
func (r *snowflakeDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	return r.c.QueryUntilNotEmpty(ctx, query)
}

// Close closes the underlying sql.DB connection.
func (r *snowflakeDriver) Close() {
	r.c.Close()
}

func (r *snowflakeDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return r.c.ColumnsFromQuery(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE
			table_schema=UPPER('%s') AND
			table_name=UPPER('%s')
		ORDER BY ordinal_position
		`, opts.Schema, opts.Table)
}

// Structure returns the layout of the database. This represents the
// "schema" with all the tables and views. Note that ordering is not
// done here. The ordering is done in the lua frontend.
func (r *snowflakeDriver) Structure() ([]*core.Structure, error) {
	query := `
		SELECT
		table_schema AS schema_name
		, table_name
		, CASE
		  WHEN table_type = 'BASE TABLE' THEN 'TABLE'
		  ELSE table_type
		  END AS table_type
		FROM
			information_schema.tables
		WHERE
			table_schema NOT IN ('INFORMATION_SCHEMA', 'PG_CATALOG')
			AND table_catalog = CURRENT_DATABASE();
	`

	rows, err := r.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return getPGStructure(rows)
}

func (r *snowflakeDriver) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT
			CURRENT_DATABASE() AS database_name
		UNION ALL
		SELECT
			DATABASE_NAME AS database_name
		FROM INFORMATION_SCHEMA.databases
		WHERE DATABASE_NAME != CURRENT_DATABASE()
		  AND IS_TRANSIENT = 'NO';
`

	rows, err := r.Query(context.Background(), query)
	if err != nil {
		return "", nil, err
	}

	first := true
	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return "", nil, err
		}
		if first {
			first = false
			current = row[0].(string)
			continue
		}
		available = append(available, row[0].(string))
	}

	return current, available, nil
}

func (r *snowflakeDriver) SelectDatabase(name string) error {
	r.connectionURL.Path = fmt.Sprintf("/%s", name)
	db, err := sql.Open("snowflake", r.connectionURL.String())
	if err != nil {
		return fmt.Errorf("unable to switch databases: %w", err)
	}
	r.c.Swap(db)
	return nil
}
