package clients

import (
	"context"
	"database/sql"
	"fmt"
	nurl "net/url"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/integratedauth/krb5"
)

// Register client
func init() {
	c := func(url string) (conn.Client, error) {
		return NewSQLServer(url)
	}
	_ = Store.Register("sqlserver", c)
}

type SQLServerClient struct {
	c   *common.Client
	url *nurl.URL
}

func NewSQLServer(url string) (*SQLServerClient, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w: ", err)
	}

	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to sqlserver database: %v", err)
	}

	return &SQLServerClient{
		c:   common.NewClient(db),
		url: u,
	}, nil
}

func (c *SQLServerClient) Query(ctx context.Context, query string) (models.IterResult, error) {
	con, err := c.c.Conn(ctx)
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

	rows, err := con.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	h, err := rows.Header()
	if err != nil {
		return nil, err
	}
	if len(h) > 0 {
		rows.SetCallback(cb)
		return rows, nil
	}
	rows.Close()

	// empty header means no result -> get affected rows
	rows, err = con.Query(ctx, "select @@ROWCOUNT as 'Rows Affected'")
	rows.SetCallback(cb)
	return rows, err
}

func (c *SQLServerClient) Layout() ([]models.Layout, error) {
	query := `SELECT table_schema, table_name FROM INFORMATION_SCHEMA.TABLES`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	children := make(map[string][]models.Layout)

	for {
		row, err := rows.Next()
		if row == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		schema := row[0].(string)
		table := row[1].(string)

		children[schema] = append(children[schema], models.Layout{
			Name:   table,
			Schema: schema,
			// TODO:
			Database: "",
			Type:     models.LayoutTypeTable,
		})

	}

	var layout []models.Layout

	for k, v := range children {
		layout = append(layout, models.Layout{
			Name:   k,
			Schema: k,
			// TODO:
			Database: "",
			Type:     models.LayoutTypeNone,
			Children: v,
		})
	}

	return layout, nil
}

func (c *SQLServerClient) Close() {
	c.c.Close()
}

func (c *SQLServerClient) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT DB_NAME(), name
		FROM sys.databases
		WHERE name != DB_NAME();
	`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return "", nil, err
	}

	for {
		row, err := rows.Next()
		if row == nil {
			break
		}
		if err != nil {
			return "", nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		current = row[0].(string)
		available = append(available, row[1].(string))
	}

	return current, available, nil
}

func (c *SQLServerClient) SelectDatabase(name string) error {
	q := c.url.Query()
	q.Set("database", name)
	c.url.RawQuery = q.Encode()

	db, err := sql.Open("sqlserver", c.url.String())
	if err != nil {
		return fmt.Errorf("unable to switch databases: %w", err)
	}

	c.c.Swap(db)

	return nil
}
