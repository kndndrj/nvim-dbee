package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/go-sql-driver/mysql"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	c := func(url string) (core.Driver, error) {
		return NewMysql(url)
	}
	_ = register(c, "mysql")
}

var _ core.Driver = (*MySQL)(nil)

type MySQL struct {
	sql *builders.Client
}

func NewMysql(url string) (*MySQL, error) {
	// add multiple statements support parameter
	match, err := regexp.MatchString(`[\?][\w]+=[\w-]+`, url)
	if err != nil {
		return nil, err
	}
	sep := "?"
	if match {
		sep = "&"
	}

	db, err := sql.Open("mysql", url+sep+"multiStatements=true")
	if err != nil {
		return nil, fmt.Errorf("unable to connect to mysql database: %v", err)
	}

	return &MySQL{
		sql: builders.NewClient(db),
	}, nil
}

func (c *MySQL) Query(ctx context.Context, query string) (core.ResultStream, error) {
	con, err := c.sql.Conn(ctx)
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
	rows, err = con.Query(ctx, "select ROW_COUNT() as 'Rows Affected'")
	rows.SetCallback(cb)
	return rows, err
}

func (c *MySQL) Structure() ([]*core.Structure, error) {
	query := `SELECT table_schema, table_name FROM information_schema.tables`

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

	var structure []*core.Structure

	for k, v := range children {
		structure = append(structure, &core.Structure{
			Name:     k,
			Schema:   k,
			Type:     core.StructureTypeNone,
			Children: v,
		})
	}

	return structure, nil
}

func (c *MySQL) Close() {
	c.sql.Close()
}
