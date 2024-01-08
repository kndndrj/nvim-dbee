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
	con, err := c.c.Conn(ctx)
	if err != nil {
		return nil, err
	}
	cb := func() {
		con.Close()
	}
	defer func() {
		if err != nil {
			cb()
		}
	}()

	rows, err := con.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	rows.SetCallback(cb)
	return rows, nil
}

// TODO(ms):
func (c *duckDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return nil, nil
}

func (c *duckDriver) Structure() ([]*core.Structure, error) {
	query := `SHOW TABLES;`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	var schema []*core.Structure
	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}

		// We know for a fact there is only one string field (see query above)
		table := row[0].(string)
		schema = append(schema, &core.Structure{
			Name:   table,
			Schema: "",
			Type:   core.StructureTypeTable,
		})
	}

	return schema, nil
}

func (c *duckDriver) Close() {
	c.c.Close()
}
