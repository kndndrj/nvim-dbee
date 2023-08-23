package clients

import (
	"database/sql"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var redshiftClient = "redshift"

func init() {
	c := func(url string) (conn.Client, error) {
		return NewRedshift(url)
	}
	_ = Store.Register(redshiftClient, c)
}

// RedshiftClient is a client for Redshift database.
// Similar to PostgresClient, but with different connection string
// and layout query (e.g. doesn't have information_schema and pg_views).
type RedshiftClient struct {
	c *common.Client
}

func NewRedshift(url string) (*RedshiftClient, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to redshift database: %v", err)
	}

	return &RedshiftClient{
		c: common.NewClient(db),
	}, nil
}

func (c *RedshiftClient) Query(query string) (models.IterResult, error) {
	con, err := c.c.Conn()
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

	rows, err := con.Query(query)
	if err != nil {
		return nil, err
	}
	rows.SetCallback(cb)
	return rows, nil
}

func (c *RedshiftClient) Close() {
	c.c.Close()
}

func (c *RedshiftClient) Layout() ([]models.Layout, error) {
	query := `
	SELECT
    trim(n.nspname) AS schema_name,
    trim(c.relname) AS table_name,
    CASE
        WHEN c.relkind = 'v' THEN 'VIEW'
        ELSE 'TABLE'
    END AS table_type
FROM
    pg_class AS c
JOIN
    pg_namespace AS n ON c.relnamespace = n.oid
WHERE
    n.nspname NOT IN ('information_schema', 'pg_catalog')
ORDER BY
    schema_name,
    table_name;
	`

	rows, err := c.Query(query)
	if err != nil {
		return nil, err
	}

	return fetchPsqlLayouts(rows, redshiftClient)
}

func fetchPsqlLayouts(rows models.IterResult, dbType string) ([]models.Layout, error) {
	children := make(map[string][]models.Layout)

	for {
		row, err := rows.Next()
		// break here to close the while loop. All layout nodes found.
		if row == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		schema, table := row[0].(string), row[1].(string)
		if dbType == "redshift" {
			typ := row[2].(string)
			children[schema] = append(children[schema], models.Layout{
				Name:     table,
				Schema:   schema,
				Database: dbType,
				Type:     getLayoutType(typ),
			})
			continue
		}
		children[schema] = append(children[schema], models.Layout{
			Name:     table,
			Schema:   schema,
			Database: dbType,
			Type:     models.LayoutTable,
		})
	}

	var layout []models.Layout

	for k, v := range children {
		layout = append(layout, models.Layout{
			Name:     k,
			Schema:   k,
			Database: dbType,
			Type:     models.LayoutNone,
			Children: v,
		})
	}

	return layout, nil
}

func getLayoutType(typ string) models.LayoutType {
	switch typ {
	case "TABLE":
		return models.LayoutTable
	case "VIEW":
		return models.LayoutView
	default:
		return models.LayoutNone
	}
}
