package adapters

import (
	"context"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var _ core.Driver = (*oracleDriver)(nil)

type oracleDriver struct {
	c *builders.Client
}

func (c *oracleDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
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

	// Remove the trailing semicolon from the query - for some reason it isn't supported in go_ora
	query = strings.TrimSuffix(query, ";")

	// Use Exec or Query depending on the query
	action := strings.ToLower(strings.Split(query, " ")[0])
	hasReturnValues := strings.Contains(strings.ToLower(query), " returning ")
	if (action == "update" || action == "delete" || action == "insert") && !hasReturnValues {
		rows, err := con.Exec(ctx, query)
		if err != nil {
			return nil, err
		}
		rows.SetCallback(cb)
		return rows, nil
	}

	rows, err := con.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(rows.Header()) == 0 {
		rows.SetCustomHeader(core.Header{"No Results"})
	}
	rows.SetCallback(cb)

	return rows, nil
}

func (c *oracleDriver) Structure() ([]*core.Structure, error) {
	query := `
		SELECT T.owner, T.table_name
		FROM (
			SELECT owner, table_name
			FROM all_tables
			UNION SELECT owner, view_name AS "table_name"
			FROM all_views
		) T
		JOIN all_users U ON T.owner = U.username
		WHERE U.common = 'NO'
		ORDER BY T.table_name
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

func (c *oracleDriver) Close() {
	c.c.Close()
}
