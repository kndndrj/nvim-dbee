package builders

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// default sql client used by other specific implementations
type Client struct {
	db             *sql.DB
	typeProcessors map[string]func(any) any
}

func NewClient(db *sql.DB, opts ...ClientOption) *Client {
	config := clientConfig{
		typeProcessors: make(map[string]func(any) any),
	}
	for _, opt := range opts {
		opt(&config)
	}

	return &Client{
		db:             db,
		typeProcessors: config.typeProcessors,
	}
}

func (c *Client) Conn(ctx context.Context) (*Conn, error) {
	conn, err := c.db.Conn(ctx)

	return &Conn{
		conn:           conn,
		typeProcessors: c.typeProcessors,
	}, err
}

// ColumnsFromQuery executes a given query on a new connection and
// converts the results to columns. A query should return a result that is
// at least 2 columns wide and have the following structure:
//
//	1st elem: name - string
//	2nd elem: type - string
//
// Query is sprintf-ed with args, so ColumnsFromQuery("select a from %s", "table_name") works.
func (c *Client) ColumnsFromQuery(query string, args ...any) ([]*core.Column, error) {
	conn, err := c.Conn(context.Background())
	if err != nil {
		return nil, err
	}

	result, err := conn.Query(context.Background(), fmt.Sprintf(query, args...))
	if err != nil {
		return nil, err
	}

	return ColumnsFromResultStream(result)
}

func (c *Client) Close() {
	c.db.Close()
}

func (c *Client) Swap(db *sql.DB) {
	c.db.Close()
	c.db = db
}

// connection to use for execution
type Conn struct {
	conn           *sql.Conn
	typeProcessors map[string]func(any) any
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

// Exec executes a query and returns a stream with single row (number of affected results).
func (c *Conn) Exec(ctx context.Context, query string) (*ResultStream, error) {
	res, err := c.conn.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	rows := NewResultStreamBuilder().
		WithNextFunc(NextSingle(affected)).
		WithHeader(core.Header{"Rows Affected"}).
		Build()

	return rows, nil
}

func (c *Conn) getTypeProcessor(typ string) func(any) any {
	proc, ok := c.typeProcessors[strings.ToLower(typ)]
	if ok {
		return proc
	}

	return func(val any) any {
		valb, ok := val.([]byte)
		if ok {
			return string(valb)
		}
		return val
	}
}

// Query executes a query on a connection and returns a result stream.
func (c *Conn) Query(ctx context.Context, query string) (*ResultStream, error) {
	dbRows, err := c.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	// create new rows
	header, err := dbRows.Columns()
	if err != nil {
		return nil, err
	}

	hasNextFunc := func() bool {
		// TODO: do we even support multiple result sets?
		// if not next result, check for any new sets
		if !dbRows.Next() {
			if !dbRows.NextResultSet() {
				return false
			}
			return dbRows.Next()
		}
		return true
	}

	nextFunc := func() (core.Row, error) {
		dbCols, err := dbRows.ColumnTypes()
		if err != nil {
			return nil, err
		}

		columns := make([]any, len(dbCols))
		columnPointers := make([]any, len(dbCols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := dbRows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		row := make(core.Row, len(dbCols))
		for i := range dbCols {
			val := *columnPointers[i].(*any)

			proc := c.getTypeProcessor(dbCols[i].DatabaseTypeName())

			row[i] = proc(val)
		}

		return row, nil
	}

	rows := NewResultStreamBuilder().
		WithNextFunc(nextFunc, hasNextFunc).
		WithHeader(header).
		WithCloseFunc(func() {
			_ = dbRows.Close()
		}).
		Build()

	return rows, nil
}
