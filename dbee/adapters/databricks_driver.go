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
	c *builders.Client
	// connectionURL is the parsed connection URL.
	// a DSN structure in the format of:
	// token:[my_token]@[hostname]:[port]/[endpoint http path]?param=value
	connectionURL *url.URL
	// currentCatalog is the current catalog.
	// Retrieved from the connectionURL parameters.
	currentCatalog string
}

// Query executes the given query and returns the result stream.
func (d *databricksDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	return d.c.QueryUntilNotEmpty(ctx, query)
}

// Columns returns the columns for the given table.
func (c *databricksDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return c.c.ColumnsFromQuery(`
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
WHERE table_catalog = '%s';
`, d.currentCatalog)

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
	case "TABLE", "BASE TABLE", "SYSTEM TABLE":
		return core.StructureTypeTable
	case "VIEW", "SYSTEM VIEW":
		return core.StructureTypeView
	case "MATERIALIZED_VIEW":
		return core.StructureTypeMaterializedView
	case "STREAMING_TABLE":
		return core.StructureTypeStreamingTable
	case "MANAGED":
		return core.StructureTypeManaged
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
func (c *databricksDriver) SelectDatabase(name string) error {
	// update the connection url with the new catalog param
	q := c.connectionURL.Query()
	q.Set("catalog", name)
	c.connectionURL.RawQuery = q.Encode()

	db, err := sql.Open("databricks", c.connectionURL.String())
	if err != nil {
		return fmt.Errorf("error switching catalog: %w", err)
	}

	// update the current catalog
	c.currentCatalog = name
	c.c.Swap(db)

	return nil
}
