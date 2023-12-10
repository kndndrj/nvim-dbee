//go:build (darwin && (amd64 || arm64)) || (freebsd && (386 || amd64 || arm || arm64)) || (linux && (386 || amd64 || arm || arm64 || ppc64le || riscv64 || s390x)) || (netbsd && amd64) || (openbsd && (amd64 || arm64)) || (windows && (amd64 || arm64))

package adapters

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&SQLite{}, "sqlite", "sqlite3")
}

var _ core.Adapter = (*SQLite)(nil)

type SQLite struct{}

func (s *SQLite) Connect(url string) (core.Driver, error) {
	db, err := sql.Open("sqlite", url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to sqlite database: %v", err)
	}

	return &sqliteDriver{
		c: builders.NewClient(db),
	}, nil
}

var _ core.Driver = (*sqliteDriver)(nil)

type sqliteDriver struct {
	c *builders.Client
}

func (c *sqliteDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
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
	rows, err = con.Query(ctx, "select changes() as 'Rows Affected'")
	rows.SetCallback(cb)
	return rows, err
}

func (c *sqliteDriver) Structure() ([]*core.Structure, error) {
	query := `SELECT name FROM sqlite_schema WHERE type ='table'`

	rows, err := c.Query(context.TODO(), query)
	if err != nil {
		return nil, err
	}

	var schema []*core.Structure
	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, err
		}

		// We know for a fact there is only one string field (see query above)
		table := row[0].(string)
		schema = append(schema, &core.Structure{
			Name:   table,
			Schema: "",
			Type:   core.StructureTypeTable,
		})
	}

	return schema, nil
}

func (c *sqliteDriver) Close() {
	c.c.Close()
}
