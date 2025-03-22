package adapters

import (
	"database/sql"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	"github.com/snowflakedb/gosnowflake"
)

func init() {
	_ = register(&Snowflake{}, "snowflake")
}

var _ core.Adapter = (*Snowflake)(nil)

type Snowflake struct{}

// Snowflake expects the connection string in dsn format.
// user[:password]@account/database/schema[?param1=value1&paramN=valueN]
// or
// user[:password]@account/database[?param1=value1&paramN=valueN]
// or
// user[:password]@host:port/database/schema?account=user_account[?param1=value1&paramN=valueN]
// or
// host:port/database/schema?account=user_account[?param1=value1&paramN=valueN]
// https://github.com/snowflakedb/gosnowflake/blob/b034584aa6fc171c1fa02e5af1f98234f24538fe/dsn.go#L308-#L314
func (r *Snowflake) Connect(rawURL string) (core.Driver, error) {
	config, err := gosnowflake.ParseDSN(rawURL)
	if err != nil {
		return nil, err
	}
	connector := gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, *config)
	db := sql.OpenDB(connector)
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &snowflakeDriver{
		c:      builders.NewClient(db),
		config: config,
	}, nil
}

func (r *Snowflake) GetHelpers(opts *core.TableOptions) map[string]string {
	list := fmt.Sprintf("SELECT * FROM %q.%q LIMIT 100;", opts.Schema, opts.Table)
	out := map[string]string{
		"List": list,
	}

	return out
}
