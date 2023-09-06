package clients

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// init registers the RedshiftClient to the store,
// i.e. to lua frontend.
func init() {
	c := func(url string) (conn.Client, error) {
		return NewRedshift(url)
	}
	_ = Store.Register("redshift", c)
}

// RedshiftClient is a sql client for Redshift.
// Mainly uses the postgres driver under the hood but with
// custom Layout function to get the table and view names correctly.
type RedshiftClient struct {
	c *common.Client
}

// NewRedshift creates a new RedshiftClient.
func NewRedshift(rawURL string) (*RedshiftClient, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w: ", err)
	}

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to postgres database: %w", err)
	}

	return &RedshiftClient{
		c: common.NewClient(db),
	}, nil
}

// Query executes a query and returns the result as an IterResult.
func (c *RedshiftClient) Query(query string) (models.IterResult, error) {
	con, err := c.c.Conn()
	if err != nil {
		return nil, err
	}
	cb := func() {
		con.Close()
	}
	defer func() {
		if err != nil {
			cb()
		}
	}()

	rows, err := con.Query(query)
	if err != nil {
		return nil, err
	}
	rows.SetCallback(cb)
	return rows, nil
}

// Close closes the underlying sql.DB connection.
func (c *RedshiftClient) Close() {
	// TODO: perhaps worth check err return statement here.
	c.c.Close()
}

// Layout returns the layout of the database. This represents the
// "schema" with all the tables and views. Note that ordering is not
// done here. The ordering is done in the lua frontend.
func (c *RedshiftClient) Layout() ([]models.Layout, error) {
	query := `
		SELECT
		trim(n.nspname) AS schema_name
		, trim(c.relname) AS table_name
		, CASE
			WHEN c.relkind = 'v' THEN 'VIEW'
			ELSE 'TABLE'
			END AS table_type
			FROM
					pg_class AS c
			INNER JOIN
					pg_namespace AS n ON c.relnamespace = n.oid
			WHERE
					n.nspname NOT IN ('information_schema', 'pg_catalog');
	`

	rows, err := c.Query(query)
	if err != nil {
		return nil, err
	}

	return getPGLayouts(rows)
}
