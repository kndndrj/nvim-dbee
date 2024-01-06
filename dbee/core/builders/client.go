package builders

import (
	"context"
	"database/sql"
	"strings"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// default sql client used by other specific implementations
type Client struct {
	db             *sql.DB
	typeProcessors map[string]func(any) any
}

type clientConfig struct {
	typeProcessors map[string]func(any) any
}

type clientOption func(*clientConfig)

func WithCustomTypeProcessor(typ string, fn func(any) any) clientOption {
	return func(cc *clientConfig) {
		t := strings.ToLower(typ)
		_, ok := cc.typeProcessors[t]
		if ok {
			// processor already registered for this type
			return
		}

		cc.typeProcessors[t] = fn
	}
}

func NewClient(db *sql.DB, opts ...clientOption) *Client {
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
