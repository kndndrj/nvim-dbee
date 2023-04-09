package clients

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
	_ "github.com/lib/pq"
)

type PostgresClient struct {
	sql *sqlClient
}

func NewPostgres(url string) (*PostgresClient, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}

	return &PostgresClient{
		sql: newSql(db),
	}, nil
}

func (c *PostgresClient) Query(query string) (conn.IterResult, error) {

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

	action := strings.ToLower(strings.Split(query, " ")[0])
	hasReturnValues := strings.Contains(strings.ToLower(query), " returning ")

	if (action == "update" || action == "delete" || action == "insert") && !hasReturnValues {
		rows, err := con.exec(query)
		if err != nil {
			return nil, err
		}
		rows.SetCallback(cb)
		return rows, nil
	}

	rows, err := con.query(query)
	if err != nil {
		return nil, err
	}
	h, err := rows.Header()
	if err != nil {
		return nil, err
	}
	if len(h) == 0 {
		rows.SetCustomHeader(conn.Header{"No Results"})
	}
	rows.SetCallback(cb)

	return rows, nil
}

func (c *PostgresClient) Schema() (conn.Schema, error) {
	query := `
		SELECT table_schema, table_name FROM information_schema.tables UNION ALL
		SELECT schemaname, matviewname FROM pg_matviews;
	`

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

func (c *PostgresClient) Close() {
	c.sql.close()
}
