package clients

import (
	"database/sql"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	_ "modernc.org/sqlite"
)

type SqliteClient struct {
	c *common.Client
}

func NewSqlite(url string) (*SqliteClient, error) {

	db, err := sql.Open("sqlite", url)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}

	return &SqliteClient{
		c: common.NewClient(db),
	}, nil
}

func (c *SqliteClient) Query(query string) (conn.IterResult, error) {

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
	rows, err = con.Query("select changes() as 'Rows Affected'")
	rows.SetCallback(cb)
	return rows, err
}

func (c *SqliteClient) Layout() ([]conn.Layout, error) {
	query := `SELECT name FROM sqlite_schema WHERE type ='table'`

	rows, err := c.Query(query)
	if err != nil {
		return nil, err
	}

	var schema []conn.Layout
	for {
		row, err := rows.Next()
		if row == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		// We know for a fact there is only one string field (see query above)
		table := row[0].(string)
		schema = append(schema, conn.Layout{
			Name:   table,
			Schema: "",
			// TODO:
			Database: "",
			Type:     conn.LayoutTable,
		})
	}

	return schema, nil
}

func (c *SqliteClient) Close() {
	c.c.Close()
}
