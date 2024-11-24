package adapters

import (
	"context"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var _ core.Driver = (*HanaDriver)(nil)

type HanaDriver struct {
	c *builders.Client
}

func (c *HanaDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	query = strings.TrimSuffix(query, ";")

	// Use Exec or Query depending on the query
	action := strings.ToLower(strings.Split(query, " ")[0])
	hasReturnValues := strings.Contains(strings.ToLower(query), " returning ")
	if (action == "update" || action == "delete" || action == "insert") && !hasReturnValues {
		return c.c.Exec(ctx, query)
	}

	return c.c.QueryUntilNotEmpty(ctx, query)
}

func (c *HanaDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return c.c.ColumnsFromQuery(`
        SELECT COLUMN_NAME, DATA_TYPE_NAME FROM SYS.TABLE_COLUMNS
        WHERE SCHEMA_NAME = '%s' AND TABLE_NAME = '%s'
    `,
		opts.Schema,
		opts.Table)
}

func (c *HanaDriver) Structure() ([]*core.Structure, error) {
	query := `
        SELECT SCHEMA_NAME, TABLE_NAME FROM SYS.TABLES
	`

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

func (c *HanaDriver) Close() {
	c.c.Close()
}
