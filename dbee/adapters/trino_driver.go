package adapters

import (
	"context" // Use database/sql types where applicable, like NullString
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	"github.com/trinodb/trino-go-client/trino"
)

var _ core.Driver = (*trinoDriver)(nil)

// Add DatabaseSwitcher if implemented
// var _ core.DatabaseSwitcher = (*trinoDriver)(nil)

type trinoDriver struct {
	client        *sql.DB
	cfg           *trino.Config
	connectionURL *url.URL
}

// Query executes a query using the Trino client.
func (d *trinoDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	rows, err := d.client.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("trino query failed: %w", err)
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	streamBuilder := builders.NewResultStreamBuilder()
	streamBuilder.WithHeader(cols)

	values := make([]interface{}, len(cols))
	scanArgs := make([]interface{}, len(cols))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	streamBuilder.WithNextFunc(
		func() (core.Row, error) {
			if rows.Next() {
				if err := rows.Scan(scanArgs...); err != nil {
					return nil, err
				}
				return values, nil
			}
			return nil, rows.Err()
		},
		rows.Next,
	)

	return streamBuilder.Build(), nil
}

// Structure retrieves the database structure (Catalogs -> Schemas -> Tables/Views).
func (d *trinoDriver) Structure() ([]*core.Structure, error) {
	ctx := context.Background()
	var catalogs []*core.Structure

	// 1. Get Catalogs
	catalogRows, err := d.queryToSlice(ctx, "SHOW CATALOGS")
	if err != nil {
		return nil, fmt.Errorf("failed to show catalogs: %w", err)
	}

	for _, row := range catalogRows {
		if len(row) == 0 {
			continue
		}
		catalogName, ok := row[0].(string)
		if !ok {
			continue
		}

		// 2. Get Schemas for each catalog
		schemaQuery := fmt.Sprintf(`SHOW SCHEMAS FROM "%s"`, catalogName)
		schemaRows, err := d.queryToSlice(ctx, schemaQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to show schemas from catalog %s: %w", catalogName, err)
		}

		var schemas []*core.Structure
		for _, schemaRow := range schemaRows {
			if len(schemaRow) == 0 {
				continue
			}
			schemaName, ok := schemaRow[0].(string)
			if !ok {
				continue
			}

			// Skip information_schema
			if schemaName == "information_schema" {
				continue
			}

			// 3. Get Tables/Views for each schema
			tableQuery := fmt.Sprintf(`
				SELECT table_schema, table_name, table_type
				FROM "%s".information_schema.tables
				WHERE table_schema = '%s'
				AND table_schema NOT IN ('information_schema')`,
				catalogName, schemaName)

			tableRows, err := d.queryToSlice(ctx, tableQuery)
			if err != nil {
				return nil, fmt.Errorf("failed to get tables for schema %s: %w", schemaName, err)
			}

			var tables []*core.Structure
			for _, tableRow := range tableRows {
				if len(tableRow) < 3 {
					continue
				}
				schema, ok1 := tableRow[0].(string)
				tableName, ok2 := tableRow[1].(string)
				tableType, ok3 := tableRow[2].(string)
				if !ok1 || !ok2 || !ok3 {
					continue
				}

				structType := core.StructureTypeTable
				if tableType == "VIEW" {
					structType = core.StructureTypeView
				}

				tables = append(tables, &core.Structure{
					Name:   tableName,
					Schema: schema,
					Type:   structType,
				})
			}

			schemas = append(schemas, &core.Structure{
				Name:     schemaName,
				Schema:   catalogName,
				Type:     core.StructureTypeSchema,
				Children: tables,
			})
		}

		catalogs = append(catalogs, &core.Structure{
			Name:     catalogName,
			Type:     core.StructureTypeSchema,
			Children: schemas,
		})
	}

	return catalogs, nil
}

// Columns retrieves column information for a table.
func (d *trinoDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	// Assuming opts.Schema is the Trino schema and we use the connection's default catalog.
	// If catalog switching is implemented, use the current catalog.
	catalog := d.cfg.Catalog // Get catalog from the current config
	if catalog == "" {
		return nil, fmt.Errorf("catalog is not set in the connection config")
	}
	if opts.Schema == "" {
		return nil, fmt.Errorf("schema is required in TableOptions")
	}
	if opts.Table == "" {
		return nil, fmt.Errorf("table is required in TableOptions")
	}

	// Use information_schema for reliable column info
	query := fmt.Sprintf(`
        SELECT column_name, data_type
        FROM %s.information_schema.columns
        WHERE table_catalog = ? AND table_schema = ? AND table_name = ?
        ORDER BY ordinal_position`, trinoQuoteIdentifier(catalog))

	rows, err := d.client.QueryContext(context.Background(), query, catalog, opts.Schema, opts.Table)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	var columns []*core.Column
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		columns = append(columns, &core.Column{
			Name: name,
			Type: dataType,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating columns: %w", err)
	}

	return columns, nil
}

// Close cleans up the driver resources.
func (d *trinoDriver) Close() {
	// The trino-go-client doesn't seem to have an explicit Close method on the client itself.
	// Connections might be managed internally or per-query. Check library docs if cleanup is needed.
}

// --- Helper Functions ---

// queryToSlice executes a query and returns all rows as a slice. Used internally for metadata.
func (d *trinoDriver) queryToSlice(ctx context.Context, query string) ([][]interface{}, error) {
	var results [][]interface{}

	rows, err := d.client.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(cols))
	scanArgs := make([]interface{}, len(cols))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		row := make([]interface{}, len(cols))
		copy(row, values)
		results = append(results, row)
	}

	return results, rows.Err()
}

// getTrinoStructureType maps Trino table types to core.StructureType.
func getTrinoStructureType(typ string) core.StructureType {
	upperType := strings.ToUpper(typ)
	switch upperType {
	case "BASE TABLE", "TABLE":
		return core.StructureTypeTable
	case "VIEW":
		return core.StructureTypeView
	case "MATERIALIZED VIEW":
		return core.StructureTypeMaterializedView
	default:
		fmt.Printf("Warning: Unknown Trino structure type: %s\n", typ)
		return core.StructureTypeNone
	}
}

// trinoQuoteIdentifier quotes an identifier if necessary (basic version).
// Trino uses double quotes for quoting.
func trinoQuoteIdentifier(name string) string {
	// Simple check: if it contains non-standard characters or is a reserved word (not checked here)
	// A robust implementation would check against reserved words and specific character rules.
	if strings.ContainsAny(name, " .-@") || strings.ToLower(name) == "table" /* add more reserved */ {
		return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`)) // Escape double quotes
	}
	return name // Return as is if simple
}

// --- DatabaseSwitcher Implementation (Optional) ---
// Trino switching usually involves changing catalog/schema in the config and potentially reconnecting.

/*
// ListDatabases lists available catalogs. Trino doesn't have a single "database" concept like PG/MySQL.
// It returns the current catalog (if set) and all available catalogs.
func (d *trinoDriver) ListDatabases() (current string, available []string, err error) {
	current = d.cfg.Catalog // Get current catalog from config

	rows, err := d.queryToSlice(context.Background(), "SHOW CATALOGS")
	if err != nil {
		return current, nil, fmt.Errorf("failed to list catalogs: %w", err)
	}

	for _, row := range rows {
		if len(row) > 0 {
			if cat, ok := row[0].(string); ok {
				available = append(available, cat)
			}
		}
	}
	return current, available, nil
}

// SelectDatabase attempts to switch the catalog.
// This likely requires creating a new client with updated config, similar to Databricks.
func (d *trinoDriver) SelectDatabase(name string) error {
	// Create new config based on old one, changing only the catalog
	newCfg := *d.cfg // Shallow copy, careful with pointers if any
	newCfg.Catalog = name
	// Schema might need resetting or explicit handling too
	// newCfg.Schema = "" // Or maybe keep the old schema? Depends on desired behavior.

	// Update the connection URL query parameter as well for consistency?
	q := d.connectionURL.Query()
	q.Set("catalog", name)
	// q.Del("schema") // Reset schema?
	d.connectionURL.RawQuery = q.Encode()
	newCfg.HTTPClient.Transport = trino.NewTransport(&newCfg) // Recreate transport? Check client docs.

	// Create a new client with the updated config
	newClient, err := trino.NewClient(&newCfg)
	if err != nil {
		return fmt.Errorf("failed to create new trino client for catalog %s: %w", name, err)
	}

	// Test the new connection? Trino client doesn't have Ping. Maybe run a simple query.
	_, testErr := newClient.Query(context.Background(), "SELECT 1", func(*trino.QueryResults) error { return nil }, func(*trino.QueryResults) error { return nil })
	if testErr != nil {
		return fmt.Errorf("failed to test connection to catalog %s: %w", name, testErr)
	}

	// Swap the client and config in the driver
	// Note: Closing the old client isn't explicitly needed based on trino-go-client docs.
	d.client = newClient
	d.cfg = &newCfg

	return nil
}

*/
