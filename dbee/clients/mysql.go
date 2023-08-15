package clients

import (
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// Register client
func init() {
	c := func(url string) (conn.Client, error) {
		return NewMysql(url)
	}
	_ = Store.Register("mysql", c)
}

type MysqlClient struct {
	sql *common.Client
}

func NewMysql(url string) (*MysqlClient, error) {
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

	return &MysqlClient{
		sql: common.NewClient(db),
	}, nil
}

func (c *MysqlClient) Query(query string) (models.IterResult, error) {
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

func (c *MysqlClient) Layout() ([]models.Layout, error) {
	query := `SELECT table_schema, table_name FROM information_schema.tables`

	rows, err := c.Query(query)
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

func (c *MysqlClient) Close() {
	c.sql.Close()
}
