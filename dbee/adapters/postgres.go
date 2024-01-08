package adapters

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	nurl "net/url"

	_ "github.com/lib/pq"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&Postgres{}, "postgres", "postgresql", "pg")

	// register special json response with gob
	gob.Register(&postgresJSONResponse{})
}

var _ core.Adapter = (*Postgres)(nil)

type Postgres struct{}

func (p *Postgres) Connect(url string) (core.Driver, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w: ", err)
	}

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to postgres database: %w", err)
	}

	jsonProcessor := func(a any) any {
		b, ok := a.([]byte)
		if !ok {
			return a
		}

		return newPostgresJSONResponse(b)
	}

	return &postgresDriver{
		c: builders.NewClient(db,
			builders.WithCustomTypeProcessor("json", jsonProcessor),
			builders.WithCustomTypeProcessor("jsonb", jsonProcessor),
		),
		url: u,
	}, nil
}

func (*Postgres) GetHelpers(opts *core.TableOptions) map[string]string {
	basicConstraintQuery := `
	SELECT tc.constraint_name, tc.table_name, kcu.column_name, ccu.table_name AS foreign_table_name, ccu.column_name AS foreign_column_name, rc.update_rule, rc.delete_rule
	FROM
		information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.referential_constraints as rc
			ON tc.constraint_name = rc.constraint_name
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
	`

	return map[string]string{
		"List":    fmt.Sprintf("SELECT * FROM %q.%q LIMIT 500", opts.Schema, opts.Table),
		"Columns": fmt.Sprintf("SELECT * FROM information_schema.columns WHERE table_name='%s' AND table_schema='%s'", opts.Table, opts.Schema),
		"Indexes": fmt.Sprintf("SELECT * FROM pg_indexes WHERE tablename='%s' AND schemaname='%s'", opts.Table, opts.Schema),
		"Foreign Keys": fmt.Sprintf("%s WHERE constraint_type = 'FOREIGN KEY' AND tc.table_name = '%s' AND tc.table_schema = '%s'",
			basicConstraintQuery,
			opts.Table,
			opts.Schema,
		),
		"References": fmt.Sprintf("%s WHERE constraint_type = 'FOREIGN KEY' AND ccu.table_name = '%s' AND tc.table_schema = '%s'",
			basicConstraintQuery,
			opts.Table,
			opts.Schema,
		),
		"Primary Keys": fmt.Sprintf("%s WHERE constraint_type = 'PRIMARY KEY' AND tc.table_name = '%s' AND tc.table_schema = '%s'",
			basicConstraintQuery,
			opts.Table,
			opts.Schema,
		),
	}
}
