//go:build cgo && ((darwin && (amd64 || arm64)) || (linux && (amd64 || arm64 || riscv64)))

package adapters

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	c := func(url string) (core.Driver, error) {
		return NewDuck(url)
	}
	_ = register(c, "duck", "duckdb")
}

var _ core.Driver = (*Duck)(nil)

type Duck struct {
	c *builders.Client
}

func NewDuck(url string) (*Duck, error) {
	db, err := sql.Open("duckdb", url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to duckdb database: %v", err)
	}

	return &Duck{
		c: builders.NewClient(db),
	}, nil
}

func (c *Duck) Query(ctx context.Context, query string) (core.ResultStream, error) {
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
	rows.SetCallback(cb)
	return rows, nil
}

func (c *Duck) Structure() ([]*core.Structure, error) {
	query := `SHOW TABLES;`

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

func (c *Duck) Close() {
	c.c.Close()
}
