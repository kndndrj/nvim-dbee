package adapters

import (
	"context"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*duckDriver)(nil)
	_ core.DatabaseSwitcher = (*duckDriver)(nil)
)

type duckDriver struct {
	c              *builders.Client
	currentDB string
}

func (d *duckDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	return d.c.QueryUntilNotEmpty(ctx, query)
}

func (d *duckDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return d.c.ColumnsFromQuery("DESCRIBE %q.%q", opts.Schema, opts.Table)
}

func (d *duckDriver) Structure() ([]*core.Structure, error) {
	catalogQuery := fmt.Sprintf(`
		SELECT table_schema, table_name, table_type
		FROM information_schema.tables
		WHERE table_catalog = '%s';`,
		d.currentDB)

	rows, err := d.Query(context.Background(), catalogQuery)
	if err != nil {
		return nil, err
	}

	return core.GetGenericStructure(rows, getDuckDBStructureType)
}

// getDuckDBStructureType returns the core.StructureType based on the
// given type string for duckdb adapter.
func getDuckDBStructureType(typ string) core.StructureType {
	// TODO: (phdah) Add more types if exists
	switch typ {
	case "BASE TABLE":
		return core.StructureTypeTable
	case "VIEW":
		return core.StructureTypeView
	default:
		return core.StructureTypeNone
	}
}

// ListDatabases returns the current catalog and a list of available catalogs.
// NOTE: (phdah) As of now, swapping catalogs is not enabled and only the
// current will be shown
func (d *duckDriver) ListDatabases() (current string, available []string, err error) {
	// no-op
	return d.currentDB, []string{"not supported yet"}, nil
}

// SelectDatabase switches the current database/catalog to the selected one.
func (d *duckDriver) SelectDatabase(name string) error {
	return nil
}

// Close closes the connection to the database.
func (d *duckDriver) Close() {
	d.c.Close()
}
