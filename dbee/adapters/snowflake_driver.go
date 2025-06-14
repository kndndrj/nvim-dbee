package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/snowflakedb/gosnowflake"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*snowflakeDriver)(nil)
	_ core.DatabaseSwitcher = (*snowflakeDriver)(nil)
)

// snowflakeDriver is a sql client for snowflakeDriver.
type snowflakeDriver struct {
	c      *builders.Client
	config gosnowflake.Config
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
    desc table %s.%s type = columns
	`, opts.Schema, opts.Table)
}

func getSnowflakeStructure(rows core.ResultStream) ([]*core.Structure, error) {
	children := make(map[string][]*core.Structure)

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}
		if len(row) < 5 {
			return nil, errors.New("could not retrieve structure: insufficient info")
		}

		table, tableType, schema := row[1].(string), row[2].(string), row[4].(string)

		if schema == "INFORMATION_SCHEMA" {
			continue
		}

		children[schema] = append(children[schema], &core.Structure{
			Name:   table,
			Schema: schema,
			Type:   getPGStructureType(tableType),
		})
	}

	structure := make([]*core.Structure, 0, len(children))

	for schema, models := range children {
		structure = append(structure, &core.Structure{
			Name:     schema,
			Schema:   schema,
			Type:     core.StructureTypeNone,
			Children: models,
		})
	}

	return structure, nil
}

// Structure returns the layout of the database. This represents the
// "schema" with all the tables and views. Note that ordering is not
// done here. The ordering is done in the lua frontend.
func (r *snowflakeDriver) Structure() ([]*core.Structure, error) {
	query := "show terse objects;"
	rows, err := r.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return getSnowflakeStructure(rows)
}

func (r *snowflakeDriver) ListDatabases() (current string, available []string, err error) {
	query := "show databases;"

	rows, err := r.Query(context.Background(), query)
	if err != nil {
		return "", nil, err
	}

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return "", nil, err
		}
		databaseName := row[2].(string)
		if databaseName == r.config.Database {
			continue
		}
		available = append(available, databaseName)
	}

	return r.config.Database, available, nil
}

func (r *snowflakeDriver) SelectDatabase(name string) error {
	config := r.config
	config.Database = name
	connector := gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, config)
	db := sql.OpenDB(connector)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("unable to ping snowflake: %w", err)
	}
	r.c.Swap(db)
	r.config = config
	return nil
}
