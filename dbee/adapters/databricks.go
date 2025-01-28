package adapters

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/databricks/databricks-sql-go"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&Databricks{}, "databricks")
}

var _ core.Adapter = (*Databricks)(nil)

type Databricks struct{}

// Connect parses the connectionURL and returns a new core.Driver
// connectionURL is a DSN structure in the format of:
//
// token:[my_token]@[hostname]:[port]/[endpoint http path]?param=value
//
// requires the 'catalog' parameter to be set.

// TODO: This could be extended with databricks connect by looking up
// the config if connectionURL is empty. Added in the future

// see https://github.com/databricks/databricks-sql-go for more information.
func (d *Databricks) Connect(connectionURL string) (core.Driver, error) {
	parsedURL, err := url.Parse(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w: ", err)
	}

	// NOTE: we could add a PingContext with timeout here but I'll leave that
	// up to the user to add in the DSN URL (given databricks bootup time).
	db, err := sql.Open("databricks", parsedURL.String())
	if err != nil {
		return nil, fmt.Errorf("invalid databricks connection string: %w", err)
	}

	currentCatalog := parsedURL.Query().Get("catalog")
	if currentCatalog == "" {
		return nil, fmt.Errorf("required parameter '?catalog=<catalog>' is missing")
	}

	return &databricksDriver{
		c:              builders.NewClient(db),
		connectionURL:  parsedURL,
		currentCatalog: currentCatalog,
	}, nil
}

// GetHelpers returns a map of helper queries for the given table.
func (d *Databricks) GetHelpers(opts *core.TableOptions) map[string]string {
	// TODO: extend this to include more helper queries
	list := fmt.Sprintf("SELECT * FROM %s.%s LIMIT 100;", opts.Schema, opts.Table)
	columns := fmt.Sprintf(`
		SELECT *
		FROM information_schema.column
		WHERE table_schema = '%s'
			AND table_name = '%s';`,
		opts.Schema, opts.Table)
	describe := fmt.Sprintf("DESCRIBE EXTENDED %s.%s;", opts.Schema, opts.Table)
	constraints := fmt.Sprintf(`
		SELECT *
		FROM information_schema.table_constraints
		WHERE table_schema = '%s'
			AND table_name = '%s';`,
		opts.Schema, opts.Table)
	keys := fmt.Sprintf(`
		SELECT *
		FROM information_schema.key_column_usage
		WHERE table_schema = '%s'
			AND table_name = '%s';`,
		opts.Schema, opts.Table)
	return map[string]string{
		"List":        list,
		"Columns":     columns,
		"Describe":    describe,
		"Constraints": constraints,
		"Keys":        keys,
	}
}
