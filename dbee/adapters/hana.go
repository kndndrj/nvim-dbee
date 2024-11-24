package adapters

import (
	"database/sql"
	"fmt"

	_ "github.com/SAP/go-hdb/driver"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

func init() {
	_ = register(&Hana{}, "hana")
}

var _ core.Adapter = (*Hana)(nil)

type Hana struct{}

func (p *Hana) Connect(url string) (core.Driver, error) {
	db, err := sql.Open("hdb", url)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	return &HanaDriver{
		c: builders.NewClient(db),
	}, nil
}

func (*Hana) GetHelpers(opts *core.TableOptions) map[string]string {
	return map[string]string{
		"List": fmt.Sprintf(
			"SELECT * FROM %q.%q LIMIT 500",
			opts.Schema, opts.Table,
		),
		"Columns": fmt.Sprintf(
			"SELECT * FROM SYS.TABLE_COLUMNS WHERE SCHEMA_NAME='%s' AND TABLE_NAME = '%s' ",
			opts.Schema, opts.Table,
		),
		"Info": fmt.Sprintf(
			"SELECT * FROM SYS.TABLES WHERE SCHEMA_NAME = '%s' AND TABLE_NAME = '%s'",
			opts.Schema, opts.Table,
		),
		"Indexes": fmt.Sprintf(
			"SELECT * FROM SYS.INDEXES WHERE TABLE_NAME = '%s' AND SCHEMA_NAME = '%s'",
			opts.Schema, opts.Table,
		),
		"Constraints": fmt.Sprintf(
			"SELECT * FROM SYS.CONSTRAINTS WHERE SCHEMA_NAME='%s' AND TABLE_NAME= '%s'",
			opts.Schema, opts.Table,
		),
	}
}
