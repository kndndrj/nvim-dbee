package common

import (
	"context"
	"database/sql"
	"time"

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

func (c *Client) Conn() (*Conn, error) {
	conn, err := c.db.Conn(context.TODO())

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

func (c *Conn) Exec(query string) (*Result, error) {
	res, err := c.conn.ExecContext(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	// create new rows
	first := true

	rows := NewResultBuilder().
		WithNextFunc(func() (models.Row, error) {
			if !first {
				return nil, nil
			}
			first = false

			affected, err := res.RowsAffected()
			if err != nil {
				return nil, err
			}
			return models.Row{affected}, nil
		}).
		WithHeader(models.Header{"Rows Affected"}).
		WithMeta(models.Meta{
			Query:     query,
			Timestamp: time.Now(),
		}).
		Build()

	return rows, nil
}

func (c *Conn) Query(query string) (*Result, error) {
	dbRows, err := c.conn.QueryContext(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	// create new rows
	header, err := dbRows.Columns()
	if err != nil {
		return nil, err
	}

	rows := NewResultBuilder().
		WithNextFunc(func() (models.Row, error) {
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
		}).
		WithHeader(header).
		WithCloseFunc(func() {
			dbRows.Close()
		}).
		WithMeta(models.Meta{
			Query:     query,
			Timestamp: time.Now(),
		}).
		Build()

	return rows, nil
}
