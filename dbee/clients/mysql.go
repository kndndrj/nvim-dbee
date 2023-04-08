package clients

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"

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

	db, err := sql.Open("mysql", url)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}

	return &MysqlClient{
		sql: newSql(db),
	}, nil
}

func (c *MysqlClient) Query(query string) (conn.IterResult, error) {

	con, err := c.sql.conn()
	if err != nil {
		return nil, err
	}

	rows, err := con.query(query)
	if err != nil {
		return nil, err
	}

	h, err := rows.Header()
	if err != nil {
		return nil, err
	}
	if len(h) > 0 {
		return rows, nil
	}

	// empty header means no result -> get affected rows
	return con.query("select ROW_COUNT() as 'Rows Affected'")
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
