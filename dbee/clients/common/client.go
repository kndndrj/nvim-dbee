package common

import (
	"context"
	"database/sql"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// connTimeOut is the timeout for a connection.
const connTimeOut = 5 * time.Minute

// Client represents the SQL client used by specific implementations.
type Client struct {
	db *sql.DB
}

// NewClient creates a new SQL client.
func NewClient(db *sql.DB) *Client {
	return &Client{
		db: db,
	}
}

// Close closes the client.
func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) Swap(db *sql.DB) {
	c.db.Close()
	c.db = db
}

// Conn represents a connection to use for execution.
type Conn struct {
	conn *sql.Conn
}

// NewConn creates a new connection to DB
// unless the context is canceled or timed out after 5 min.
func (c *Client) NewConn(ctx context.Context) (*Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeOut)
	defer cancel()

	conn, err := c.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	return &Conn{
		conn: conn,
	}, nil
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

// Exec executes a query and returns the result.
func (c *Conn) Exec(ctx context.Context, query string) (*Result, error) {
	res, err := c.conn.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}

	// Create new rows
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

// Query executes a query and returns the result.
func (c *Conn) Query(ctx context.Context, query string) (*Result, error) {
	dbRows, err := c.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	// Create new rows
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

			// TODO: Do we even support multiple result sets?
			// If not, check for any new sets.
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

			columns := make([]interface{}, len(dbCols))
			columnPointers := make([]interface{}, len(dbCols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := dbRows.Scan(columnPointers...); err != nil {
				return nil, err
			}

			row := make(models.Row, len(dbCols))
			for i := range dbCols {
				val := columns[i]
				// TODO: This breaks some types with some drivers (namely SQL Server newid()).
				// Add a generic way of doing this with ResultBuilder.
				// Fix for some strings being interpreted as bytes.
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
