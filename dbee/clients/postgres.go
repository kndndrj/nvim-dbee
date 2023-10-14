package clients

import (
	"context"
	"database/sql"
	"fmt"
	nurl "net/url"
	"strings"

	_ "github.com/lib/pq"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// Register client
func init() {
	c := func(url string) (conn.Client, error) {
		return NewPostgres(url)
	}
	_ = Store.Register("postgres", c)
}

type PostgresClient struct {
	c   *common.Client
	url *nurl.URL
}

func NewPostgres(url string) (*PostgresClient, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w: ", err)
	}

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to postgres database: %w", err)
	}

	return &PostgresClient{
		c:   common.NewClient(db),
		url: u,
	}, nil
}

func (c *PostgresClient) Query(ctx context.Context, query string) (models.IterResult, error) {
	con, err := c.c.NewConn(ctx)
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

	action := strings.ToLower(strings.Split(query, " ")[0])
	hasReturnValues := strings.Contains(strings.ToLower(query), " returning ")

	if (action == "update" || action == "delete" || action == "insert") && !hasReturnValues {
		rows, err := con.Exec(ctx, query)
		if err != nil {
			return nil, err
		}
		rows.SetCallback(cb)
		return rows, nil
	}

	rows, err := con.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	h, err := rows.Header()
	if err != nil {
		return nil, err
	}
	if len(h) == 0 {
		rows.SetCustomHeader(models.Header{"No Results"})
	}
	rows.SetCallback(cb)

	return rows, nil
}

func (c *PostgresClient) Layout() ([]models.Layout, error) {
	query := `
		SELECT table_schema, table_name, table_type FROM information_schema.tables UNION ALL
		SELECT schemaname, matviewname, 'VIEW' FROM pg_matviews;
	`

	ctx := context.Background()
	rows, err := c.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	return getPGLayouts(rows)
}

func (c *PostgresClient) Close() {
	c.c.Close()
}

func (c *PostgresClient) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT current_database(), datname FROM pg_database
		WHERE datistemplate = false
		AND datname != current_database();
	`

	ctx := context.Background()
	rows, err := c.Query(ctx, query)
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

func (c *PostgresClient) SelectDatabase(name string) error {
	c.url.Path = fmt.Sprintf("/%s", name)
	db, err := sql.Open("postgres", c.url.String())
	if err != nil {
		return fmt.Errorf("unable to switch databases: %w", err)
	}

	c.c.Swap(db)

	return nil
}

// getPGLayouts fetches the layout from the postgres database.
// rows is at least 3 column wide result
func getPGLayouts(rows models.IterResult) ([]models.Layout, error) {
	children := make(map[string][]models.Layout)

	for {
		row, err := rows.Next()
		// break here to close the while loop. All layout nodes found.
		if row == nil {
			break
		}
		if err != nil {
			return nil, err
		}

		schema, table, tableType := row[0].(string), row[1].(string), row[2].(string)

		children[schema] = append(children[schema], models.Layout{
			Name:   table,
			Schema: schema,
			Type:   getPGLayoutType(tableType),
		})
	}

	var layout []models.Layout

	for k, v := range children {
		layout = append(layout, models.Layout{
			Name:     k,
			Schema:   k,
			Type:     models.LayoutTypeNone,
			Children: v,
		})
	}

	return layout, nil
}

// getPGLayoutType returns the layout type based on the string.
func getPGLayoutType(typ string) models.LayoutType {
	switch typ {
	case "TABLE", "BASE TABLE":
		return models.LayoutTypeTable
	case "VIEW":
		return models.LayoutTypeView
	default:
		return models.LayoutTypeNone
	}
}
