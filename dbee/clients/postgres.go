package clients

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	_ "github.com/lib/pq"
)

type PostgresClient struct {
	c *common.Client
}

func NewPostgres(url string) (*PostgresClient, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}

	return &PostgresClient{
		c: common.NewClient(db),
	}, nil
}

func (c *PostgresClient) Query(query string) (models.IterResult, error) {

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

	action := strings.ToLower(strings.Split(query, " ")[0])
	hasReturnValues := strings.Contains(strings.ToLower(query), " returning ")

	if (action == "update" || action == "delete" || action == "insert") && !hasReturnValues {
		rows, err := con.Exec(query)
		if err != nil {
			return nil, err
		}
		rows.SetCallback(cb)
		return rows, nil
	}

	rows, err := con.Query(query)
	if err != nil {
		return nil, err
	}
	h, err := rows.Header()
	if err != nil {
		return nil, err
	}
	if len(h) == 0 {
		rows.SetCustomHeader(models.Header{"No Results"})
	}
	rows.SetCallback(cb)

	return rows, nil
}

func (c *PostgresClient) Layout() ([]models.Layout, error) {
	query := `
		SELECT table_schema, table_name FROM information_schema.tables UNION ALL
		SELECT schemaname, matviewname FROM pg_matviews;
	`

	rows, err := c.Query(query)
	if err != nil {
		return nil, err
	}

	children := make(map[string][]models.Layout)

	for {
		row, err := rows.Next()
		if row == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		schema := row[0].(string)
		table := row[1].(string)

		children[schema] = append(children[schema], models.Layout{
			Name:   table,
			Schema: schema,
			// TODO:
			Database: "",
			Type:     models.LayoutTable,
		})
	}

	var layout []models.Layout

	for k, v := range children {
		layout = append(layout, models.Layout{
			Name:   k,
			Schema: k,
			// TODO:
			Database: "",
			Type:     models.LayoutNone,
			Children: v,
		})
	}

	return layout, nil
}

func (c *PostgresClient) Close() {
	c.c.Close()
}
