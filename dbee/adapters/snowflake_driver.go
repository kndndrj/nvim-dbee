package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
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
	c       *builders.Client
	config  gosnowflake.Config
	cacheMu sync.Mutex

	structureCache []*core.Structure
	databaseCache  []string
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
	rows, err := r.Query(context.Background(), "show columns in table "+snowflakeTableName(opts.Schema, opts.Table))
	if err != nil {
		return nil, err
	}
	return getSnowflakeColumns(rows)
}

func getSnowflakeColumns(rows core.ResultStream) ([]*core.Column, error) {
	columns := []*core.Column{}

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}
		if len(row) < 4 {
			continue
		}

		name, ok := row[2].(string)
		if !ok {
			continue
		}
		typ, ok := row[3].(string)
		if !ok {
			continue
		}

		columns = append(columns, &core.Column{Name: name, Type: typ})
	}

	return columns, nil
}

func getSnowflakeStructure(rows core.ResultStream) ([]*core.Structure, error) {
	children := make(map[string][]*core.Structure)

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}
		if len(row) < 5 {
			continue
		}

		table, ok := row[1].(string)
		if !ok {
			continue
		}
		tableType, ok := row[2].(string)
		if !ok {
			continue
		}
		schema, ok := row[4].(string)
		if !ok || schema == "INFORMATION_SCHEMA" {
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
	r.cacheMu.Lock()
	if r.structureCache != nil {
		defer r.cacheMu.Unlock()
		return r.structureCache, nil
	}
	r.cacheMu.Unlock()

	if r.config.Database == "" {
		return nil, nil
	}

	query := "show terse objects in database " + snowflakeIdentifier(r.config.Database)
	if r.config.Schema != "" {
		query = "show terse objects in schema " + snowflakeTableName(r.config.Database, r.config.Schema)
	}
	rows, err := r.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	structure, err := getSnowflakeStructure(rows)
	if err != nil {
		return nil, err
	}

	r.cacheMu.Lock()
	r.structureCache = structure
	r.cacheMu.Unlock()

	return structure, nil
}

func (r *snowflakeDriver) ListDatabases() (current string, available []string, err error) {
	r.cacheMu.Lock()
	if r.databaseCache != nil {
		defer r.cacheMu.Unlock()
		return r.currentDatabaseLabel(), r.databaseCache, nil
	}
	r.cacheMu.Unlock()

	rows, err := r.Query(context.Background(), "show databases;")
	if err != nil {
		return "", nil, err
	}

	header := rows.Header()
	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return "", nil, err
		}
		databaseName := snowflakeDatabaseNameFromRow(header, row)
		if databaseName == "" {
			continue
		}
		if strings.EqualFold(databaseName, r.config.Database) {
			continue
		}
		available = append(available, databaseName)
	}

	r.cacheMu.Lock()
	r.databaseCache = available
	r.cacheMu.Unlock()

	return r.currentDatabaseLabel(), available, nil
}

func (r *snowflakeDriver) SelectDatabase(name string) error {
	config := r.config
	config.Database = name
	connector := gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, config)
	db := sql.OpenDB(connector)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("unable to ping snowflake: %w", err)
	}
	r.c.Swap(db)
	r.config = config
	r.cacheMu.Lock()
	r.structureCache = nil
	r.cacheMu.Unlock()
	return nil
}

func (r *snowflakeDriver) currentDatabaseLabel() string {
	if r.config.Database != "" {
		return r.config.Database
	}
	return "Databases"
}

func snowflakeTableName(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		out = append(out, snowflakeIdentifier(part))
	}
	return strings.Join(out, ".")
}

func snowflakeIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func snowflakeDatabaseNameFromRow(header core.Header, row core.Row) string {
	for i, column := range header {
		if strings.EqualFold(column, "name") && i < len(row) {
			return snowflakeString(row[i])
		}
	}
	if len(row) > 1 {
		return snowflakeString(row[1])
	}
	return ""
}

func snowflakeString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprint(v)
	}
}
