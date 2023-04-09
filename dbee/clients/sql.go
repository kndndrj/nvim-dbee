package clients

import (
	"context"
	"database/sql"
	"sync"
	"time"

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

func (c *sqlClient) conn() (*sqlConn, error) {
	conn, err := c.db.Conn(context.TODO())

	return &sqlConn{
		conn: conn,
	}, err
}

func (c *sqlClient) close() {
	c.db.Close()
}

// connection to use for execution
type sqlConn struct {
	conn *sql.Conn
}

func (c *sqlConn) exec(query string) (*SqlExecRows, error) {
	res, err := c.conn.ExecContext(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	meta := conn.Meta{
		Query:     query,
		Timestamp: time.Now(),
	}

	rows := newSqlExecRows(res, meta)

	return rows, nil
}

func (c *sqlConn) query(query string) (*SqlQueryRows, error) {
	dbRows, err := c.conn.QueryContext(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	meta := conn.Meta{
		Query:     query,
		Timestamp: time.Now(),
	}

	rows := newSqlQueryRows(dbRows, meta)

	return rows, nil
}

func (c *sqlConn) close() error {
	return c.conn.Close()
}

// rows returned by sql.exec
type SqlExecRows struct {
	// first is reset after the first call to Next()
	first        bool
	dbResult     sql.Result
	meta         conn.Meta
	customHeader conn.Header
	callback     func()
	once         sync.Once
}

func newSqlExecRows(result sql.Result, meta conn.Meta) *SqlExecRows {
	return &SqlExecRows{
		dbResult: result,
		meta:     meta,
		first:    true,
		once: sync.Once{},
	}
}

func (r *SqlExecRows) SetCustomHeader(header conn.Header) {
	r.customHeader = header
}

func (r *SqlExecRows) SetCallback(callback func()) {
	r.callback = callback
}

func (r *SqlExecRows) Meta() (conn.Meta, error) {
	return r.meta, nil
}

func (r *SqlExecRows) Header() (conn.Header, error) {
	if len(r.customHeader) > 0 {
		return r.customHeader, nil
	}
	return conn.Header{"Rows Affected"}, nil
}

func (r *SqlExecRows) Next() (conn.Row, error) {
	affected, err := r.dbResult.RowsAffected()
	if err != nil {
		return nil, err
	}

	if r.first {
		r.first = false
		return conn.Row{affected}, nil
	}
	return nil, nil
}

func (r *SqlExecRows) Close() {
	if r.callback != nil {
		r.once.Do(r.callback)
	}
}

type SqlQueryRows struct {
	dbRows       *sql.Rows
	meta         conn.Meta
	customHeader conn.Header
	callback     func()
	once         sync.Once
}

// rows returned by sql.query
func newSqlQueryRows(rows *sql.Rows, meta conn.Meta) *SqlQueryRows {
	return &SqlQueryRows{
		dbRows: rows,
		meta:   meta,
		once: sync.Once{},
	}
}

func (r *SqlQueryRows) SetCustomHeader(header conn.Header) {
	r.customHeader = header
}

func (r *SqlQueryRows) SetCallback(callback func()) {
	r.callback = callback
}

func (r *SqlQueryRows) Meta() (conn.Meta, error) {
	return r.meta, nil
}

func (r *SqlQueryRows) Header() (conn.Header, error) {
	if len(r.customHeader) > 0 {
		return r.customHeader, nil
	}
	return r.dbRows.Columns()
}

func (r *SqlQueryRows) Next() (conn.Row, error) {
	var err error
	var row conn.Row
	defer func() {
		if err != nil || row == nil {
			r.Close()
		}
	}()

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

	columns := make([]any, len(dbCols))
	columnPointers := make([]any, len(dbCols))
	for i := range columns {
		columnPointers[i] = &columns[i]
	}

	if err := r.dbRows.Scan(columnPointers...); err != nil {
		return nil, err
	}

	row = make(conn.Row, len(dbCols))
	for i := range dbCols {
		val := *columnPointers[i].(*any)
		// fix for some strings being interpreted as bytes
		valb, ok := val.([]byte)
		if ok {
			val = string(valb)
		}
		row[i] = val
	}

	return row, nil
}

func (r *SqlQueryRows) Close() {
	r.dbRows.Close()
	if r.callback != nil {
		r.once.Do(r.callback)
	}
}
