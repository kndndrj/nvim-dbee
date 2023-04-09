package clients

import (
	"database/sql"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
	_ "modernc.org/sqlite"
)

type SqliteClient struct {
	sql *sqlClient
}

func NewSqlite(url string) (*SqliteClient, error) {

	db, err := sql.Open("sqlite", url)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}

	return &SqliteClient{
		sql: newSql(db),
	}, nil
}

func (c *SqliteClient) Query(query string) (conn.IterResult, error) {

	con, err := c.sql.conn()
	if err != nil {
		return nil, err
	}
	cb := func() {
		con.close()
	}
	defer func() {
		if err != nil {
			cb()
		}
	}()

	rows, err := con.query(query)
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
	rows, err = con.query("select changes() as 'Rows Affected'")
	rows.SetCallback(cb)
	return rows, err
}

func (c *SqliteClient) Schema() (conn.Schema, error) {
	query := `SELECT name FROM sqlite_schema WHERE type ='table'`

	rows, err := c.Query(query)
	if err != nil {
		return nil, err
	}

	var tables []string
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
		tables = append(tables, table)
	}

	return conn.Schema{
		"tables": tables,
	}, nil
}

func (c *SqliteClient) Close() {
	c.sql.close()
}
