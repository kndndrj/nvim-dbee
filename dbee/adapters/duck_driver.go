package adapters

import (
	"context"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var _ core.Driver = (*duckDriver)(nil)

type duckDriver struct {
	c *builders.Client
}

func (c *duckDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	return c.c.QueryUntilNotEmpty(ctx, query)
}

func (c *duckDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return c.c.ColumnsFromQuery("DESCRIBE %q.%q", opts.Schema, opts.Table)
}

func (c *duckDriver) Structure() ([]*core.Structure, error) {
	catalogQuery := `
		SELECT table_schema, table_name, table_type
		FROM information_schema.tables;`

	rows, err := c.Query(context.Background(), catalogQuery)
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

// Close closes the connection to the database.
func (c *duckDriver) Close() {
	c.c.Close()
}
