package adapters

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql" // Import normally to access mysql.ParseDSN

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
	// parse the connection string into a mysql.Config struct
	cfg, err := mysql.ParseDSN(url)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w", err)
	}

	// add multiple statements support parameter
	cfg.MultiStatements = true

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to mysql database: %v", err)
	}

	return &mySQLDriver{
		c:   builders.NewClient(db),
		cfg: cfg,
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
