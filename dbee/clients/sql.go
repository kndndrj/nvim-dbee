package clients

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
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

func (c *sqlConn) exec(query string) (*SqlRows, error) {
	res, err := c.conn.ExecContext(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	// create new rows
	first := true

	rows := newSqlRowsBuilder().
		WithNextFunc(func() (conn.Row, error) {
			if !first {
				return nil, nil
			}
			first = false

			affected, err := res.RowsAffected()
			if err != nil {
				return nil, err
			}
			return conn.Row{affected}, nil

		}).
		WithHeader(conn.Header{"Rows Affected"}).
		WithMeta(conn.Meta{
			Query:     query,
			Timestamp: time.Now(),
		}).
		Build()

	return rows, nil
}

func (c *sqlConn) query(query string) (*SqlRows, error) {
	dbRows, err := c.conn.QueryContext(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	// create new rows
	header, err := dbRows.Columns()
	if err != nil {
		return nil, err
	}

	rows := newSqlRowsBuilder().
		WithNextFunc(func() (conn.Row, error) {
			dbCols, err := dbRows.Columns()
			if err != nil {
				return nil, err
			}

			// TODO: do we even support multiple result sets?
			// if not next result, check for any new sets
			if !dbRows.Next() {
				if !dbRows.NextResultSet() {
					return nil, nil
				}
				dbCols, err = dbRows.Columns()
				if err != nil {
					return nil, err
				}
				if !dbRows.Next() {
					return nil, nil
				}
			}

			columns := make([]any, len(dbCols))
			columnPointers := make([]any, len(dbCols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := dbRows.Scan(columnPointers...); err != nil {
				return nil, err
			}

			row := make(conn.Row, len(dbCols))
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
		}).
		WithHeader(header).
		WithCloseFunc(func() {
			dbRows.Close()
		}).
		WithMeta(conn.Meta{
			Query:     query,
			Timestamp: time.Now(),
		}).
		Build()

	return rows, nil
}

func (c *sqlConn) close() error {
	return c.conn.Close()
}

// SqlRows fills conn.IterResult interface for all sql dbs
type SqlRows struct {
	next     func() (conn.Row, error)
	header   conn.Header
	close    func()
	meta     conn.Meta
	callback func()
	once     sync.Once
}

func (r *SqlRows) SetCustomHeader(header conn.Header) {
	r.header = header
}

func (r *SqlRows) SetCallback(callback func()) {
	r.callback = callback
}

func (r *SqlRows) Meta() (conn.Meta, error) {
	return r.meta, nil
}

func (r *SqlRows) Header() (conn.Header, error) {
	return r.header, nil
}

func (r *SqlRows) Next() (conn.Row, error) {
	rows, err := r.next()
	if err != nil || rows == nil {
		r.Close()
		return nil, err
	}
	return rows, nil
}

func (r *SqlRows) Close() {
	r.close()
	if r.callback != nil {
		r.once.Do(r.callback)
	}
}

// SqlRowsBuilder builds the rows
type SqlRowsBuilder struct {
	next   func() (conn.Row, error)
	header conn.Header
	close  func()
	meta   conn.Meta
}

func newSqlRowsBuilder() *SqlRowsBuilder {
	return &SqlRowsBuilder{
		next:   func() (conn.Row, error) { return nil, nil },
		header: conn.Header{},
		close:  func() {},
		meta:   conn.Meta{},
	}
}

func (b *SqlRowsBuilder) WithNextFunc(fn func() (conn.Row, error)) *SqlRowsBuilder {
	b.next = fn
	return b
}

func (b *SqlRowsBuilder) WithHeader(header conn.Header) *SqlRowsBuilder {
	b.header = header
	return b
}

func (b *SqlRowsBuilder) WithCloseFunc(fn func()) *SqlRowsBuilder {
	b.close = fn
	return b
}

func (b *SqlRowsBuilder) WithMeta(meta conn.Meta) *SqlRowsBuilder {
	b.meta = meta
	return b
}

func (b *SqlRowsBuilder) Build() *SqlRows {
	return &SqlRows{
		next:   b.next,
		header: b.header,
		close:  b.close,
		meta:   b.meta,
		once:   sync.Once{},
	}
}
