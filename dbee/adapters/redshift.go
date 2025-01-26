package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// init registers the RedshiftClient to the store,
// i.e. to lua frontend.
func init() {
	_ = register(&Redshift{}, "redshift")
}

var _ core.Adapter = (*Redshift)(nil)

type Redshift struct{}

func (r *Redshift) Connect(rawURL string) (core.Driver, error) {
	connURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// TODO: perhaps better to use something else than postgres driver..
	db, err := sql.Open("postgres", connURL.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to redshift: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping redshift: %w", err)
	}

	return &redshiftDriver{
		c:             builders.NewClient(db),
		connectionURL: connURL,
	}, nil
}

func (r *Redshift) GetHelpers(opts *core.TableOptions) map[string]string {
	out := make(map[string]string, 0)
	list := fmt.Sprintf("SELECT * FROM %q.%q LIMIT 100;", opts.Schema, opts.Table)

	switch opts.Materialization {
	case core.StructureTypeTable:
		out = map[string]string{
			"List":    list,
			"Columns": fmt.Sprintf("SELECT * FROM information_schema.columns WHERE table_name='%s' AND table_schema='%s';", opts.Table, opts.Schema),
			"Indexes": fmt.Sprintf("SELECT * FROM pg_indexes WHERE tablename='%s' AND schemaname='%s';", opts.Table, opts.Schema),
			"Foreign Keys": fmt.Sprintf(`
				SELECT tc.constraint_name, tc.table_name, kcu.column_name, ccu.table_name AS foreign_table_name, ccu.column_name AS foreign_column_name, rc.update_rule, rc.delete_rule
				FROM
					information_schema.table_constraints AS tc
					JOIN information_schema.key_column_usage AS kcu
						ON tc.constraint_name = kcu.constraint_name
					JOIN information_schema.referential_constraints as rc
						ON tc.constraint_name = rc.constraint_name
					JOIN information_schema.constraint_column_usage AS ccu
						ON ccu.constraint_name = tc.constraint_name
				WHERE constraint_type = 'FOREIGN KEY' AND tc.table_name = '%s' AND tc.table_schema = '%s';`,

				opts.Table,
				opts.Schema,
			),
			"Table Definition": fmt.Sprintf(`
				SELECT
					*
				FROM svv_table_info
				WHERE "schema" = '%s'
					AND "table" = '%s';`,

				opts.Schema,
				opts.Table,
			),
		}

	case core.StructureTypeView:
		out = map[string]string{
			"List": list,
			"View Definition": fmt.Sprintf(`
				SELECT
					*
				FROM pg_views
				WHERE schemaname = '%s'
					AND viewname = '%s';`,

				opts.Schema,
				opts.Table,
			),
		}
	}

	return out
}
