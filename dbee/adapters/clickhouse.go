package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&Clickhouse{}, "clickhouse")
}

var _ core.Adapter = (*Clickhouse)(nil)

type Clickhouse struct{}

func (p *Clickhouse) Connect(url string) (core.Driver, error) {
	options, err := clickhouse.ParseDSN(url)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w", err)
	}

	jsonProcessor := func(a any) any {
		b, ok := a.([]byte)
		if !ok {
			return a
		}

		return newPostgresJSONResponse(b)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db := clickhouse.OpenDB(options)
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("pinging connection failed with %v", err)
	}

	return &clickhouseDriver{
		c: builders.NewClient(db,
			builders.WithCustomTypeProcessor("json", jsonProcessor),
		),
		opts: options,
	}, nil
}

func (*Clickhouse) GetHelpers(opts *core.TableOptions) map[string]string {
	return map[string]string{
		"List": fmt.Sprintf(
			"SELECT * FROM %q.%q LIMIT 500",
			opts.Schema, opts.Table,
		),
		"Columns": fmt.Sprintf(
			"DESCRIBE %q.%q",
			opts.Schema, opts.Table,
		),
		"Info": fmt.Sprintf(
			"SELECT * FROM system.tables WHERE database = '%s' AND name = '%s'",
			opts.Schema, opts.Table,
		),
	}
}
