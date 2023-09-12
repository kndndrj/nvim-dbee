package common

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// default sql client used by other specific implementations
type Client struct {
	db *sql.DB
}

func NewClient(db *sql.DB) *Client {
	return &Client{
		db: db,
	}
}

func (c *Client) Conn(ctx context.Context) (*Conn, error) {
	conn, err := c.db.Conn(ctx)

	return &Conn{
		conn: conn,
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
	conn *sql.Conn
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) Exec(ctx context.Context, query string) (*Result, error) {
	res, err := c.conn.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}

	// create new rows
	first := true

	nextFn := func() (models.Row, error) {
		if !first {
			return nil, errors.New("no next row")
		}
		first = false

		affected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		return models.Row{affected}, nil
	}

	hasNextFn := func() bool {
		return !first
	}

	rows := NewResultBuilder().
		WithNextFunc(nextFn, hasNextFn).
		WithHeader(models.Header{"Rows Affected"}).
		Build()

	return rows, nil
}

func (c *Conn) Query(ctx context.Context, query string) (*Result, error) {
	dbRows, err := c.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	// create new rows
	header, err := dbRows.Columns()
	if err != nil {
		return nil, err
	}

	hasNextFn := func() bool {
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

	nextFn := func() (models.Row, error) {
		dbCols, err := dbRows.Columns()
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

		row := make(models.Row, len(dbCols))
		for i := range dbCols {
			val := *columnPointers[i].(*any)
			// TODO: this breaks some types with some drivers (namely sqlserver newid()):
			// add a generic way of doing this with ResultBuilder
			// fix for some strings being interpreted as bytes
			valb, ok := val.([]byte)
			if ok {
				val = string(valb)
			}
			row[i] = val
		}

		return row, nil
	}

	rows := NewResultBuilder().
		WithNextFunc(nextFn, hasNextFn).
		WithHeader(header).
		WithCloseFunc(func() {
			dbRows.Close()
		}).
		Build()

	return rows, nil
}
