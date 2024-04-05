package adapters

import (
	"context"
	"database/sql"
	"fmt"
	nurl "net/url"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*verticaDriver)(nil)
	_ core.DatabaseSwitcher = (*verticaDriver)(nil)
)

type verticaDriver struct {
	c   *builders.Client
	url *nurl.URL
}

func (c *verticaDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	action := strings.ToLower(strings.Split(query, " ")[0])
	hasReturnValues := strings.Contains(strings.ToLower(query), " returning ")

	if (action == "update" || action == "delete" || action == "insert") && !hasReturnValues {
		return c.c.Exec(ctx, query)
	}

	return c.c.QueryUntilNotEmpty(ctx, query)
}

func (c *verticaDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return c.c.ColumnsFromQuery(`
		SELECT column_name, data_type
		FROM v_catalog.columns
		WHERE
			table_schema='%s' AND
			table_name='%s'
		`, opts.Schema, opts.Table)
}

func (c *verticaDriver) Structure() ([]*core.Structure, error) {
	query := `
		SELECT table_schema, table_name, 'TABLE' as table_type FROM v_catalog.tables UNION ALL
		SELECT table_schema, table_name, 'VIEW' as table_type FROM v_catalog.views;
	`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	return getPGStructure(rows)
}

func (c *verticaDriver) Close() {
	c.c.Close()
}

func (c *verticaDriver) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT current_database(), database_name as datname FROM v_catalog.databases
		WHERE database_name != current_database();
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

func (c *verticaDriver) SelectDatabase(name string) error {
	c.url.Path = fmt.Sprintf("/%s", name)
	db, err := sql.Open("vertica", c.url.String())
	if err != nil {
		return fmt.Errorf("unable to switch databases: %w", err)
	}

	c.c.Swap(db)

	return nil
}
