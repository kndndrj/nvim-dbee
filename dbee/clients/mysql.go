package clients

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
)

type MysqlClient struct {
	sql *common.Client
}

func NewMysql(url string) (*MysqlClient, error) {
	// add multiple statements support parameter
	match, err := regexp.MatchString(`[\?][\w]+=[\w-]+`, url)
	if err != nil {
		log.Fatal(err)
	}
	if match {
		url = url + "&multiStatements=true"
	} else {
		url = url + "?multiStatements=true"
	}

	db, err := sql.Open("mysql", url)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}

	return &MysqlClient{
		sql: common.NewClient(db),
	}, nil
}

func (c *MysqlClient) Query(query string) (conn.IterResult, error) {

	con, err := c.sql.Conn()
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
	rows, err = con.Query("select ROW_COUNT() as 'Rows Affected'")
	rows.SetCallback(cb)
	return rows, err
}

func (c *MysqlClient) Layout() ([]conn.Layout, error) {
	query := `SELECT table_schema, table_name FROM information_schema.tables`

	rows, err := c.Query(query)
	if err != nil {
		return nil, err
	}

	children := make(map[string][]conn.Layout)

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

		children[schema] = append(children[schema], conn.Layout{
			Name:   table,
			Schema: schema,
			// TODO:
			Database: "",
			Type:     conn.LayoutTable,
		})

	}

	var layout []conn.Layout

	for k, v := range children {
		layout = append(layout, conn.Layout{
			Name:   k,
			Schema: k,
			// TODO:
			Database: "",
			Type:     conn.LayoutNone,
			Children: v,
		})
	}

	return layout, nil
}

func (c *MysqlClient) Close() {
	c.sql.Close()
}
