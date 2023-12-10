package adapters

import (
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	nurl "net/url"

	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/integratedauth/krb5"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&SQLServer{}, "sqlserver", "mssql")

	gob.Register(uuid.UUID{})
}

var _ core.Adapter = (*SQLServer)(nil)

type SQLServer struct{}

func (s *SQLServer) Connect(url string) (core.Driver, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w: ", err)
	}

	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to sqlserver database: %v", err)
	}

	return &sqlServerDriver{
		c: builders.NewClient(db,
			builders.WithCustomTypeProcessor(
				"uniqueidentifier",
				func(a any) any {
					b, ok := a.([]byte)
					if !ok {
						return a
					}

					id, err := uuid.FromBytes(b)
					if err != nil {
						return a
					}

					return id
				}),
		),
		url: u,
	}, nil
}

var (
	_ core.Driver           = (*sqlServerDriver)(nil)
	_ core.DatabaseSwitcher = (*sqlServerDriver)(nil)
)

type sqlServerDriver struct {
	c   *builders.Client
	url *nurl.URL
}

func (c *sqlServerDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
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

	if len(rows.Header()) > 0 {
		rows.SetCallback(cb)
		return rows, nil
	}
	rows.Close()

	// empty header means no result -> get affected rows
	rows, err = con.Query(ctx, "select @@ROWCOUNT as 'Rows Affected'")
	rows.SetCallback(cb)
	return rows, err
}

func (c *sqlServerDriver) Structure() ([]*core.Structure, error) {
	query := `SELECT table_schema, table_name FROM INFORMATION_SCHEMA.TABLES`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	children := make(map[string][]*core.Structure)

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}

		// We know for a fact there are 2 string fields (see query above)
		schema := row[0].(string)
		table := row[1].(string)

		children[schema] = append(children[schema], &core.Structure{
			Name:   table,
			Schema: schema,
			Type:   core.StructureTypeTable,
		})

	}

	var layout []*core.Structure

	for k, v := range children {
		layout = append(layout, &core.Structure{
			Name:     k,
			Schema:   k,
			Type:     core.StructureTypeNone,
			Children: v,
		})
	}

	return layout, nil
}

func (c *sqlServerDriver) Close() {
	c.c.Close()
}

func (c *sqlServerDriver) ListDatabases() (current string, available []string, err error) {
	query := `
		SELECT DB_NAME(), name
		FROM sys.databases
		WHERE name != DB_NAME();
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

func (c *sqlServerDriver) SelectDatabase(name string) error {
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
