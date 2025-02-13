package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

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

func getColumnIndex(header []string, column_name string) (int, bool) {
	for i, s := range header {
		if s == column_name {
			return i, true
		}
	}
	return -1, false
}

func printHeader(header []string) string {
	out := ""
	for _, s := range header {
		out = out + ", " + s
	}
	return out
}

func specificColumnsFromResultStream(rows core.ResultStream, wanted_column_names []string) ([]map[string]string, error) {
	out := []map[string]string{}

	for rows.HasNext() {
		row_meta := make(map[string]string)

		row, err := rows.Next()
		if err != nil {
			return nil, fmt.Errorf("result.Next: %w", err)
		}

		if len(row) < 2 {
			return nil, errors.New("columns in result are less than wanted columns")
		}

		for _, col_name := range wanted_column_names {
			idx, ok := getColumnIndex(rows.Header(), col_name)
			if !ok {
				return nil, errors.New("could not find column: " + col_name + "Header: " + printHeader(rows.Header()))
			}
			var value string

			switch col_name {
			case "data_type":
				parsed := make(map[string]any)
				unparsed := row[idx].([]byte)
				if !ok {
					return nil, errors.New("could not retreive column info for " + col_name + ": type not a string")
				}
				err := json.Unmarshal(unparsed, parsed)
				fmt.Sprintf("parsed json type: %T", parsed)
				if err != nil {
					return nil, errors.New("could not parse data_type map from snowflake row")
				}
				value, ok = parsed["type"].(string)
				if !ok {
					return nil, errors.New("could not retreive column info for " + col_name + ": type not a string")
				}
				value = strings.ToLower(value)
				if value == "fixed" {
					value = "numeric"
				}
				// return nil, fmt.Errorf("data_type: %s, type: %T", row[idx], row[idx])
				row_meta[col_name] = value
			default:
				value, ok := row[idx].(string)
				if !ok {
					return nil, errors.New("could not retreive column info for " + col_name + ": type not a string")
				}
				value = strings.ToLower(value)
				row_meta[col_name] = value
			}
		}
		out = append(out, row_meta)
	}

	return out, nil
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
	// return r.c.ColumnsFromQuery(`
	// 	SELECT column_name, data_type
	// 	FROM information_schema.columns
	// 	WHERE
	// 		table_schema=UPPER('%s') AND
	// 		table_name=UPPER('%s')
	// 	ORDER BY ordinal_position
	// 	`, opts.Schema, opts.Table)
	query := `show columns in %s.%s`
	result, err := r.c.Query(context.Background(), fmt.Sprintf(query, opts.Schema, opts.Table))
	if err != nil {
		return nil, err
	}

	column_map, err := specificColumnsFromResultStream(result, []string{"column_name", "data_type"})
	if err != nil {
		return nil, err
	}

	out := []*core.Column{}
	for _, meta := range column_map {
		out = append(out, &core.Column{
			Name: meta["column_name"],
			Type: meta["data_type"],
		})
	}
	return out, nil
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
