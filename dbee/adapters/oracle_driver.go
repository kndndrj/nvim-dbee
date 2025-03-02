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

func (d *oracleDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	// Remove the trailing semicolon from the query - for some reason it isn't supported in go_ora
	query = strings.TrimSuffix(query, ";")

	// Use Exec or Query depending on the query
	action := strings.ToLower(strings.Split(query, " ")[0])
	hasReturnValues := strings.Contains(strings.ToLower(query), " returning ")
	if (action == "update" || action == "delete" || action == "insert") && !hasReturnValues {
		return d.c.Exec(ctx, query)
	}

	return d.c.QueryUntilNotEmpty(ctx, query)
}

func (d *oracleDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return d.c.ColumnsFromQuery(`
		SELECT
			col.column_name,
			col.data_type
		FROM sys.all_tab_columns col
		INNER JOIN sys.all_tables t
			ON col.owner = t.owner
			AND col.table_name = t.table_name
		WHERE col.owner = '%s'
			AND col.table_name = '%s'
		ORDER BY col.owner, col.table_name, col.column_id `,

		opts.Schema,
		opts.Table)
}

func (d *oracleDriver) Structure() ([]*core.Structure, error) {
	query := `
		SELECT owner, object_name, type
		FROM (
			SELECT owner, table_name as object_name, 'TABLE' as type
			FROM all_tables
			UNION ALL
			SELECT owner, table_name as object_name, 'EXTERNAL TABLE' as type
			FROM all_external_tables
			UNION ALL
			SELECT owner, view_name as object_name, 'VIEW' as type
			FROM all_views
			UNION ALL
			SELECT owner, mview_name as object_name, 'MATERIALIZED VIEW' as type
			FROM all_mviews
		)
		WHERE owner IN (SELECT username FROM all_users WHERE common = 'NO')
		ORDER BY owner, object_name
	`

	rows, err := d.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	decodeStructureType := func(s string) core.StructureType {
		switch s {
		case "TABLE", "EXTERNAL TABLE":
			return core.StructureTypeTable
		case "VIEW":
			return core.StructureTypeView
		case "MATERIALIZED VIEW":
			return core.StructureTypeMaterializedView
		default:
			return core.StructureTypeNone
		}
	}

	return core.GetGenericStructure(rows, decodeStructureType)
}

func (d *oracleDriver) Close() { d.c.Close() }
