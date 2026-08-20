package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*trinoDriver)(nil)
	_ core.DatabaseSwitcher = (*trinoDriver)(nil)
)

type trinoDriver struct {
	c             *builders.Client
	connectionURL *url.URL
	// catalogMu guards catalog and connectionURL, which SelectDatabase mutates
	// while Structure's fan-out goroutines and Columns read them.
	catalogMu sync.RWMutex
	// catalog is the catalog currently selected in the DSN.
	catalog        string
	schemas        []string
	schemasCatalog string
}

func (d *trinoDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	return d.c.QueryUntilNotEmpty(ctx, query)
}

func (d *trinoDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return d.c.ColumnsFromQuery("DESCRIBE %s", quoteTrino(d.currentCatalog(), opts.Schema, opts.Table))
}

// maxStructureConcurrency caps the number of parallel SHOW TABLES queries.
const maxStructureConcurrency = 8

// Structure returns the schemas and tables of the current catalog.
//
// It deliberately avoids information_schema: Trino resolves it by enumerating
// per-table metadata through the connector at query time, so an unscoped query
// over a lakehouse sized catalog can run for minutes. SHOW SCHEMAS plus one
// SHOW TABLES per schema stays scoped and runs in parallel.
func (d *trinoDriver) Structure() ([]*core.Structure, error) {
	ctx := context.Background()
	catalog := d.currentCatalog()
	all, err := d.singleColumnQuery(ctx, fmt.Sprintf("SHOW SCHEMAS FROM %s", quoteTrino(catalog)))
	if err != nil {
		return nil, err
	}

	// the allowlist only applies to the catalog it was written for
	allow := d.schemas
	if catalog != d.schemasCatalog {
		allow = nil
	}

	schemas := filterSchemas(all, allow)
	tables := make([]schemaTables, len(schemas))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxStructureConcurrency)
	for i, schema := range schemas {
		g.Go(func() error {
			names, err := d.singleColumnQuery(ctx, fmt.Sprintf("SHOW TABLES FROM %s", quoteTrino(catalog, schema)))
			tables[i] = schemaTables{schema: schema, names: names, err: err}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var (
		failed  int
		lastErr error
	)
	for _, t := range tables {
		if t.err != nil {
			failed++
			lastErr = t.err
		}
	}

	// every schema failing is systemic rather than per-schema - surface it
	if failed > 0 && failed == len(schemas) {
		return nil, fmt.Errorf("could not list tables in any of the %d schemas: %w", len(schemas), lastErr)
	}

	return buildStructure(tables), nil
}

// schemaTables holds the result of one SHOW TABLES query.
type schemaTables struct {
	schema string
	names  []string
	err    error
}

// singleColumnQuery runs a query and collects its first column into a slice.
// SHOW SCHEMAS, SHOW TABLES and SHOW CATALOGS all return exactly one column.
func (d *trinoDriver) singleColumnQuery(ctx context.Context, query string) ([]string, error) {
	rows, err := d.c.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}
		if len(row) < 1 {
			continue
		}

		value, ok := row[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", row[0])
		}
		out = append(out, value)
	}

	return out, nil
}

func (d *trinoDriver) Close() {
	d.c.Close()
}

// currentCatalog returns the selected catalog under a read lock.
func (d *trinoDriver) currentCatalog() string {
	d.catalogMu.RLock()
	defer d.catalogMu.RUnlock()

	return d.catalog
}

// filterSchemas drops information_schema and applies the allowlist when one is
// set. The order of `all` is preserved, so the drawer order stays stable.
func filterSchemas(all []string, allow []string) []string {
	out := make([]string, 0, len(all))
	for _, schema := range all {
		if schema == "information_schema" {
			continue
		}
		if len(allow) > 0 && !slices.Contains(allow, schema) {
			continue
		}
		out = append(out, schema)
	}
	return out
}

// buildStructure turns per-schema table names into drawer nodes, keeping the
// order of `tables` so the drawer order stays stable across refreshes.
func buildStructure(tables []schemaTables) []*core.Structure {
	structure := make([]*core.Structure, 0, len(tables))

	for _, table := range tables {
		children := make([]*core.Structure, 0, len(table.names))
		for _, name := range table.names {
			children = append(children, &core.Structure{
				Name:   name,
				Schema: table.schema,
				Type:   core.StructureTypeTable,
			})
		}

		structure = append(structure, &core.Structure{
			Name:     table.schema,
			Schema:   table.schema,
			Type:     core.StructureTypeSchema,
			Children: children,
		})
	}
	return structure
}

func (d *trinoDriver) ListDatabases() (current string, available []string, err error) {
	catalogs, err := d.singleColumnQuery(context.Background(), "SHOW CATALOGS")
	if err != nil {
		return "", nil, err
	}
	return d.currentCatalog(), catalogs, nil
}

// SelectDatabase switches the current catalog.
// The stored URL already holds the expanded access token, so switching does not
// re-trigger authentication.
func (d *trinoDriver) SelectDatabase(name string) error {
	d.catalogMu.Lock()
	defer d.catalogMu.Unlock()

	query := d.connectionURL.Query()
	query.Set("catalog", name)
	d.connectionURL.RawQuery = query.Encode()

	db, err := sql.Open("trino", d.connectionURL.String())
	if err != nil {
		return fmt.Errorf("error switching catalog: %w", err)
	}

	d.catalog = name
	d.c.Swap(db)

	return nil
}
