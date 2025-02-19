package adapters

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*mySQLDriver)(nil)
	_ core.DatabaseSwitcher = (*mySQLDriver)(nil)
)

type mySQLDriver struct {
	c   *builders.Client
	cfg *mysql.Config
}

func (c *mySQLDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	// run query, fallback to affected rows
	return c.c.QueryUntilNotEmpty(ctx, query, "select ROW_COUNT() as 'Rows Affected'")
}

func (c *mySQLDriver) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT IFNULL(DATABASE(), 'mysql') as current_database, SCHEMA_NAME as available_databases
    FROM information_schema.SCHEMATA
    WHERE SCHEMA_NAME <> IFNULL(DATABASE(), '');
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

func (c *mySQLDriver) SelectDatabase(name string) error {
	c.cfg.DBName = name
	db, err := sql.Open("mysql", c.cfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("unable to switch databases: %w", err)
	}

	c.c.Swap(db)

	return nil
}

func (c *mySQLDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return c.c.ColumnsFromQuery("DESCRIBE `%s`.`%s`", opts.Schema, opts.Table)
}

func (c *mySQLDriver) Structure() ([]*core.Structure, error) {
	query := `SELECT table_schema, table_name FROM information_schema.tables`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	children := make(map[string][]*core.Structure)

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		schema := row[0].(string)
		table := row[1].(string)

		children[schema] = append(children[schema], &core.Structure{
			Name:   table,
			Schema: schema,
			Type:   core.StructureTypeTable,
		})

	}

	var structure []*core.Structure

	for k, v := range children {
		structure = append(structure, &core.Structure{
			Name:     k,
			Schema:   k,
			Type:     core.StructureTypeNone,
			Children: v,
		})
	}

	return structure, nil
}

func (c *mySQLDriver) Close() {
	c.c.Close()
}
