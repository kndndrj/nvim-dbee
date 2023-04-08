package clients

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
)

type MysqlClient struct {
	sql *sqlClient
}

func NewMysql(url string) (*MysqlClient, error) {
	// add multiple statements support parameter
	match, err := regexp.MatchString(`[\?][\w]+=[\w-]+`, url)
	if err != nil {
		log.Fatal(err)
	}
	if match {
		url = url + "&multiStatements=true"
	} else {
		url = url + "?multiStatements=true"
	}
	fmt.Println(url)

	db, err := sql.Open("mysql", url)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}

	return &MysqlClient{
		sql: newSql(db),
	}, nil
}

func (c *MysqlClient) Query(query string) (conn.IterResult, error) {

	dbRows, err := c.sql.query(query)
	if err != nil {
		return nil, err
	}

	meta := conn.Meta{
		Query:     query,
		Timestamp: time.Now(),
	}

	rows := newMysqlRows(dbRows, meta)

	return rows, nil
}

func (c *MysqlClient) Schema() (conn.Schema, error) {
	query := `SELECT table_schema, table_name FROM information_schema.tables`

	rows, err := c.Query(query)
	if err != nil {
		return nil, err
	}

	var schema = make(conn.Schema)

	for {
		row, err := rows.Next()
		if row == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		key := row[0].(string)
		val := row[1].(string)
		schema[key] = append(schema[key], val)
	}

	return schema, nil
}

func (c *MysqlClient) Close() {
	c.sql.close()
}

type MysqlRows struct {
	dbRows *sqlRows
	meta   conn.Meta
}

func newMysqlRows(rows *sqlRows, meta conn.Meta) *MysqlRows {
	return &MysqlRows{
		dbRows: rows,
		meta:   meta,
	}
}

func (r *MysqlRows) Meta() (conn.Meta, error) {
	return r.meta, nil
}

func (r *MysqlRows) Header() (conn.Header, error) {
	return r.dbRows.header()
}

func (r *MysqlRows) Next() (conn.Row, error) {

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

func (r *MysqlRows) Close() {
	r.dbRows.close()
}
