package common

import (
	"context"
	"database/sql"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// DatabaseClient is an interface that represents a database client.
type DatabaseClient interface {
	Conn() (DatabaseConnection, error)
	Close()
	Swap(db *sql.DB)
}

// DatabaseConnection is an interface that represents a database connection.
type DatabaseConnection interface {
	Close() error
	Exec(query string) (models.IterResult, error)
	Query(query string) (models.IterResult, error)
}

type Client struct {
	db *sql.DB
}

func NewClient(db *sql.DB) *Client {
	return &Client{
		db: db,
	}
}

func (c *Client) Conn() (DatabaseConnection, error) {
	conn, err := c.db.Conn(context.TODO())
	if err != nil {
		return nil, err
	}
	return &Connection{
		conn: conn,
	}, nil
}

func (c *Client) Close() {
	c.db.Close()
}

func (c *Client) Swap(db *sql.DB) {
	c.db.Close()
	c.db = db
}

type Connection struct {
	conn *sql.Conn
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

func (c *Connection) Exec(query string) (models.IterResult, error) {
	res, err := c.conn.ExecContext(context.TODO(), query)
	if err != nil {
		return nil, err
	}

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

func (c *Connection) Query(query string) (models.IterResult, error) {
	dbRows, err := c.conn.QueryContext(context.TODO(), query)
	if err != nil {
		return nil, err
	}

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
