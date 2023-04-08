package clients

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
	_ "github.com/lib/pq"
)

// TODO: use this as a default sql client
type PostgresClient struct {
	db *sql.DB
}

func NewPostgres(url string) (*PostgresClient, error) {
	conn, err := sql.Open("postgres", url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return nil, err
	}

	return &PostgresClient{
		db: conn,
	}, nil
}

func (c *PostgresClient) Query(query string) (conn.IterResult, error) {

	dbRows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}

	meta := conn.Meta{
		Query:     query,
		Timestamp: time.Now(),
	}

	pgRows := newPGRows(dbRows, meta)

	return pgRows, nil
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
		key := string(row[0].([]byte))
		val := string(row[1].([]byte))
		schema[key] = append(schema[key], val)
	}

	return schema, nil
}

func (c *PostgresClient) Close() {
	c.db.Close()
}

type PGRows struct {
	dbRows *sql.Rows
	meta   conn.Meta
}

func newPGRows(pgRows *sql.Rows, meta conn.Meta) *PGRows {
	return &PGRows{
		dbRows: pgRows,
		meta:   meta,
	}
}

func (r *PGRows) Meta() (conn.Meta, error) {
	return r.meta, nil
}

func (r *PGRows) Header() (conn.Header, error) {
	header, err := r.dbRows.Columns()
	if err != nil {
		return nil, err
	}
	if len(header) == 0 {
		return conn.Header{"No Results"}, nil
	}

	return header, nil
}

func (r *PGRows) Next() (conn.Row, error) {
	dbCols, err := r.dbRows.Columns()
	if err != nil {
		return nil, err
	}

	// TODO: support multiple result sets?
	// if not next result, check for any new sets
	if !r.dbRows.Next() {
		if !r.dbRows.NextResultSet() {
			return nil, nil
		}
		dbCols, err = r.dbRows.Columns()
		if err != nil {
			return nil, err
		}
		if !r.dbRows.Next() {
			return nil, nil
		}
	}

	// Create a slice of any's to represent each column,
	// and a second slice to contain pointers to each item in the columns slice.
	columns := make([]any, len(dbCols))
	columnPointers := make([]any, len(dbCols))
	for i := range columns {
		columnPointers[i] = &columns[i]
	}

	// Scan the result into the column pointers...
	if err := r.dbRows.Scan(columnPointers...); err != nil {
		return nil, err
	}

	// Create our map, and retrieve the value for each column from the pointers slice,
	// storing it in the map with the name of the column as the key.
	var row = make(conn.Row, len(dbCols))
	for i := range dbCols {
		val := columnPointers[i].(*any)
		row[i] = *val
	}

	return row, nil
}

func (r *PGRows) Close() {
	r.dbRows.Close()
}
