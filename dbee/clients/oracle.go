package clients

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	_ "github.com/sijms/go-ora/v2"
)

// Register client
func init() {
	c := func(url string) (conn.Client, error) {
		return NewOracle(url)
	}
	_ = Store.Register("oracle", c)
}

type OracleClient struct {
	c *common.Client
}

func NewOracle(url string) (*OracleClient, error) {
	db, err := sql.Open("oracle", url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to oracle database: %v", err)
	}

	return &OracleClient{
		c: common.NewClient(db),
	}, nil
}

func (c *OracleClient) Query(query string) (models.IterResult, error) {

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

	// Remove the trailing semicolon from the query - for some reason it isn't supported in go_ora
	query = strings.TrimSuffix(query, ";")

	// Use Exec or Query depending on the query
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

func (c *OracleClient) Layout() ([]models.Layout, error) {
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

func (c *OracleClient) Close() {
	c.c.Close()
}
