package clients

import (
	"context"
	"database/sql"
	"fmt"
	nurl "net/url"
	"strings"

	_ "github.com/lib/pq"

	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// Register client
func init() {
	c := func(url string) (core.Client, error) {
		return NewPostgres(url)
	}
	_ = register(c, "postgres", "postgresql", "pg")
}

var _ core.Client = (*Postgres)(nil)

type Postgres struct {
	c   *builders.Client
	url *nurl.URL
}

func NewPostgres(url string) (*Postgres, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w: ", err)
	}

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to postgres database: %w", err)
	}

	return &Postgres{
		c:   builders.NewClient(db),
		url: u,
	}, nil
}

func (c *Postgres) Query(ctx context.Context, query string) (core.IterResult, error) {
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
	if len(rows.Header()) == 0 {
		rows.SetCustomHeader(core.Header{"No Results"})
	}
	rows.SetCallback(cb)

	return rows, nil
}

func (c *Postgres) Layout() ([]core.Layout, error) {
	query := `
		SELECT table_schema, table_name, table_type FROM information_schema.tables UNION ALL
		SELECT schemaname, matviewname, 'VIEW' FROM pg_matviews;
	`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	return getPGLayouts(rows)
}

func (c *Postgres) Close() {
	c.c.Close()
}

func (c *Postgres) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT current_database(), datname FROM pg_database
		WHERE datistemplate = false
		AND datname != current_database();
	`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return "", nil, err
	}

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return "", nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		current = row[0].(string)
		available = append(available, row[1].(string))
	}

	return current, available, nil
}

func (c *Postgres) SelectDatabase(name string) error {
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
func getPGLayouts(rows core.IterResult) ([]core.Layout, error) {
	children := make(map[string][]core.Layout)

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}

		schema, table, tableType := row[0].(string), row[1].(string), row[2].(string)

		children[schema] = append(children[schema], core.Layout{
			Name:   table,
			Schema: schema,
			Type:   getPGLayoutType(tableType),
		})
	}

	var layout []core.Layout

	for k, v := range children {
		layout = append(layout, core.Layout{
			Name:     k,
			Schema:   k,
			Type:     core.LayoutTypeNone,
			Children: v,
		})
	}

	return layout, nil
}

// getPGLayoutType returns the layout type based on the string.
func getPGLayoutType(typ string) core.LayoutType {
	switch typ {
	case "TABLE", "BASE TABLE":
		return core.LayoutTypeTable
	case "VIEW":
		return core.LayoutTypeView
	default:
		return core.LayoutTypeNone
	}
}
