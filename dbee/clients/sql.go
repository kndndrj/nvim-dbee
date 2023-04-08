package clients

import (
	"database/sql"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
	_ "github.com/lib/pq"
)

// default sql client used by other specific implementations
type sqlClient struct {
	db *sql.DB
}

func newSql(db *sql.DB) *sqlClient {
	return &sqlClient{
		db: db,
	}
}

func (c *sqlClient) query(query string) (*sqlRows, error) {

	dbRows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}

	rows := newSqlRows(dbRows)

	return rows, nil
}

func (c *sqlClient) close() {
	c.db.Close()
}

type sqlRows struct {
	dbRows *sql.Rows
}

func newSqlRows(rows *sql.Rows) *sqlRows {
	return &sqlRows{
		dbRows: rows,
	}
}

func (r *sqlRows) header() (conn.Header, error) {
	header, err := r.dbRows.Columns()
	if err != nil {
		return nil, err
	}
	if len(header) == 0 {
		return conn.Header{"No Results"}, nil
	}

	return header, nil
}

func (r *sqlRows) next() (conn.Row, error) {
	dbCols, err := r.dbRows.Columns()
	if err != nil {
		return nil, err
	}

	// TODO: do we even support multiple result sets?
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

func (r *sqlRows) close() {
	r.dbRows.Close()
}
