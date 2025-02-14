package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*databricksDriver)(nil)
	_ core.DatabaseSwitcher = (*databricksDriver)(nil)
)

// databricksDriver is a driver for Databricks.
type databricksDriver struct {
	// c is the client used to execute queries.
	c              *builders.Client
	connectionURL  *url.URL
	currentCatalog string
}

// Query executes the given query and returns the result stream.
func (d *databricksDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	return d.c.QueryUntilNotEmpty(ctx, query)
}

// Columns returns the columns and their types for the given table.
func (d *databricksDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return d.c.ColumnsFromQuery(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE
			table_schema='%s' AND
			table_name='%s';`,
		opts.Schema, opts.Table)
}

// Structure returns the structure of the current catalog/database.
func (d *databricksDriver) Structure() ([]*core.Structure, error) {
	catalogQuery := fmt.Sprintf(`
		SELECT table_schema, table_name, table_type
		FROM system.information_schema.tables
		WHERE table_catalog = '%s'; `,
		d.currentCatalog)

	rows, err := d.Query(context.Background(), catalogQuery)
	if err != nil {
		return nil, err
	}

	return core.GetGenericStructure(rows, getDatabricksStructureType)
}

// getDatabricksStructureType returns the core.StructureType based on the
// given type string for databricks adapter.
func getDatabricksStructureType(typ string) core.StructureType {
	switch typ {
	case "TABLE", "BASE TABLE", "SYSTEM TABLE", "MANAGED", "STREAMING_TABLE", "MANAGED_SHALLOW_CLONE", "MANAGED_DEEP_CLONE":
		return core.StructureTypeTable
	case "VIEW", "SYSTEM VIEW", "MATERIALIZED_VIEW":
		return core.StructureTypeView
	default:
		return core.StructureTypeNone
	}
}

// Close closes the connection to the database.
func (d *databricksDriver) Close() {
	d.c.Close()
}

// ListDatabases returns the current catalog and a list of
// available catalogs.
func (d *databricksDriver) ListDatabases() (current string, available []string, err error) {
	query := `SHOW CATALOGS;`

	rows, err := d.Query(context.Background(), query)
	if err != nil {
		return "", nil, err
	}

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return "", nil, err
		}

		catalog, ok := row[0].(string)
		if !ok {
			return "", nil, fmt.Errorf("expected string, got %T", row[0])
		}
		available = append(available, catalog)
	}

	return d.currentCatalog, available, nil
}

// SelectDatabase switches the current database/catalog to the selected one.
func (d *databricksDriver) SelectDatabase(name string) error {
	// update the connection url with the new catalog param
	q := d.connectionURL.Query()
	q.Set("catalog", name)
	d.connectionURL.RawQuery = q.Encode()

	db, err := sql.Open("databricks", d.connectionURL.String())
	if err != nil {
		return fmt.Errorf("error switching catalog: %w", err)
	}

	// update the current catalog
	d.currentCatalog = name
	d.c.Swap(db)

	return nil
}
