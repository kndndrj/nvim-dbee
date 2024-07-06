package adapters

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

func init() {
	_ = register(&Snowflake{}, "snowflake")
}

var _ core.Adapter = (*Snowflake)(nil)

type Snowflake struct{}

func (r *Snowflake) Connect(rawURL string) (core.Driver, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}
	connURL := SnowflakeURL{*parsedURL}

	db, err := sql.Open("snowflake", connURL.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to snowflake: %w", err)
	}

	return &snowflakeDriver{
		c:             builders.NewClient(db),
		connectionURL: &connURL,
	}, nil
}

func (r *Snowflake) GetHelpers(opts *core.TableOptions) map[string]string {
	out := make(map[string]string, 0)
	list := fmt.Sprintf("SELECT * FROM %q.%q LIMIT 100;", opts.Schema, opts.Table)

	switch opts.Materialization {
	case core.StructureTypeTable:
		out = map[string]string{
			"List": list,
		}

	case core.StructureTypeView:
		out = map[string]string{
			"List": list,
		}
	}

	return out
}
