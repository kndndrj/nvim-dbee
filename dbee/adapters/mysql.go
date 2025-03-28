package adapters

import (
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/go-sql-driver/mysql"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&MySQL{}, "mysql")
}

var _ core.Adapter = (*MySQL)(nil)

type MySQL struct{}

func (m *MySQL) Connect(url string) (core.Driver, error) {
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

	return &mySQLDriver{
		c: builders.NewClient(db),
	}, nil
}

func (*MySQL) GetHelpers(opts *core.TableOptions) map[string]string {
	return map[string]string{
		"List":         fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT 500", opts.Schema, opts.Table),
		"Columns":      fmt.Sprintf("DESCRIBE `%s`.`%s`", opts.Schema, opts.Table),
		"Indexes":      fmt.Sprintf("SHOW INDEXES FROM `%s`.`%s`", opts.Schema, opts.Table),
		"Foreign Keys": fmt.Sprintf("SELECT * FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s' AND CONSTRAINT_TYPE = 'FOREIGN KEY'", opts.Schema, opts.Table),
		"Primary Keys": fmt.Sprintf("SELECT * FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s' AND CONSTRAINT_TYPE = 'PRIMARY KEY'", opts.Schema, opts.Table),
	}
}
