package clients

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
	_ "github.com/mattn/go-sqlite3"
)

type SqliteClient struct {
	sql *sqlClient
}

func NewSqlite(url string) (*SqliteClient, error) {

	db, err := sql.Open("sqlite3", url)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}

	return &SqliteClient{
		sql: newSql(db),
	}, nil
}

func (c *SqliteClient) Query(query string) (conn.IterResult, error) {

	dbRows, err := c.sql.query(query)
	if err != nil {
		return nil, err
	}

	meta := conn.Meta{
		Query:     query,
		Timestamp: time.Now(),
	}

	rows := newSqliteRows(dbRows, meta)

	return rows, nil
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

type SqliteRows struct {
	dbRows *sqlRows
	meta   conn.Meta
}

func newSqliteRows(rows *sqlRows, meta conn.Meta) *SqliteRows {
	return &SqliteRows{
		dbRows: rows,
		meta:   meta,
	}
}

func (r *SqliteRows) Meta() (conn.Meta, error) {
	return r.meta, nil
}

func (r *SqliteRows) Header() (conn.Header, error) {
	return r.dbRows.header()
}

func (r *SqliteRows) Next() (conn.Row, error) {

	row, err := r.dbRows.next()
	if err != nil {
		return nil, err
	}

	// fix for pq interpreting strings as bytes - hopefully does not break
	for i, val := range row {
		valb, ok := val.([]byte)
		if ok {
			val = string(valb)
		}
		row[i] = val
	}

	return row, nil
}

func (r *SqliteRows) Close() {
	r.dbRows.close()
}
