package clients

import (
	"database/sql"
	"fmt"

	_ "github.com/googleapis/go-sql-spanner"
	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

type SpannerClient struct {
	sql *common.Client
}

func NewSpanner(url string) (*SpannerClient, error) {
	db, err := sql.Open("spanner", url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to Spanner database: %w", err)
	}

	return &SpannerClient{
		sql: common.NewClient(db),
	}, nil
}

func (c *SpannerClient) Query(query string) (models.IterResult, error) {

	con, err := c.sql.Conn()
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

	h, err := rows.Header()
	if err != nil {
		return nil, err
	}
	if len(h) > 0 {
		rows.SetCallback(cb)
		return rows, nil
	}
	rows.Close()

	// empty header means no result -> get affected rows
	rows, err = con.Query("select ROW_COUNT() as 'Rows Affected'")
	rows.SetCallback(cb)
	return rows, err
}

func (c *SpannerClient) Layout() ([]models.Layout, error) {
	query := `SELECT table_schema, table_name FROM information_schema.tables`

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

	layout := make([]models.Layout, 0, len(children))

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

func (c *SpannerClient) Close() {
	c.sql.Close()
}
