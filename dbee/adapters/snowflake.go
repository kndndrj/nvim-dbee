package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
	config, err := prepareSnowflakeConfig(rawURL)
	if err != nil {
		return nil, err
	}
	connector := gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, *config)
	db := sql.OpenDB(connector)
	ctx, cancel := context.WithTimeout(context.Background(), snowflakePingTimeout(*config))
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping snowflake: %w", err)
	}
	defaultCtx, defaultCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer defaultCancel()
	hydrateSnowflakeSessionDefaults(defaultCtx, db, config)

	return &snowflakeDriver{
		c:      builders.NewClient(db),
		config: *config,
	}, nil
}

func (r *Snowflake) GetHelpers(opts *core.TableOptions) map[string]string {
	list := fmt.Sprintf("SELECT * FROM %q.%q LIMIT 100;", opts.Schema, opts.Table)
	out := map[string]string{
		"List": list,
	}

	return out
}

func snowflakePingTimeout(config gosnowflake.Config) time.Duration {
	if config.Authenticator == gosnowflake.AuthTypeOAuthAuthorizationCode || config.Authenticator == gosnowflake.AuthTypeExternalBrowser {
		return 2 * time.Minute
	}
	return 30 * time.Second
}

func hydrateSnowflakeSessionDefaults(ctx context.Context, db *sql.DB, config *gosnowflake.Config) {
	if config.Database != "" && config.Schema != "" {
		return
	}

	var currentDatabase, currentSchema sql.NullString
	err := db.QueryRowContext(ctx, "select current_database(), current_schema()").Scan(&currentDatabase, &currentSchema)
	if err != nil {
		return
	}

	applySnowflakeSessionDefaults(config, currentDatabase, currentSchema)
}

func applySnowflakeSessionDefaults(config *gosnowflake.Config, currentDatabase, currentSchema sql.NullString) {
	if config.Database == "" && currentDatabase.Valid {
		config.Database = currentDatabase.String
	}
	if config.Schema == "" && currentSchema.Valid {
		config.Schema = currentSchema.String
	}
}
