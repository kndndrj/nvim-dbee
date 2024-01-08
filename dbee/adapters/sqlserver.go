package adapters

import (
	"database/sql"
	"encoding/gob"
	"fmt"
	nurl "net/url"

	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/integratedauth/krb5"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

// Register client
func init() {
	_ = register(&SQLServer{}, "sqlserver", "mssql")

	gob.Register(uuid.UUID{})
}

var _ core.Adapter = (*SQLServer)(nil)

type SQLServer struct{}

func (s *SQLServer) Connect(url string) (core.Driver, error) {
	u, err := nurl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("could not parse db connection string: %w: ", err)
	}

	db, err := sql.Open("sqlserver", u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to sqlserver database: %v", err)
	}

	return &sqlServerDriver{
		c: builders.NewClient(db,
			builders.WithCustomTypeProcessor(
				"uniqueidentifier",
				func(a any) any {
					b, ok := a.([]byte)
					if !ok {
						return a
					}

					id, err := uuid.FromBytes(b)
					if err != nil {
						return a
					}

					return id
				}),
		),
		url: u,
	}, nil
}

func (*SQLServer) GetHelpers(opts *core.TableOptions) map[string]string {
	columnSummary := fmt.Sprintf(`
      SELECT c.column_name + ' (' +
          ISNULL(( SELECT 'PK, ' FROM information_schema.table_constraints AS k JOIN information_schema.key_column_usage AS kcu ON k.constraint_name = kcu.constraint_name WHERE constraint_type='PRIMARY KEY' AND k.table_name = c.table_name AND kcu.column_name = c.column_name), '') +
          ISNULL(( SELECT 'FK, ' FROM information_schema.table_constraints AS k JOIN information_schema.key_column_usage AS kcu ON k.constraint_name = kcu.constraint_name WHERE constraint_type='FOREIGN KEY' AND k.table_name = c.table_name AND kcu.column_name = c.column_name), '') +
          data_type + COALESCE('(' + RTRIM(CAST(character_maximum_length AS VARCHAR)) + ')','(' + RTRIM(CAST(numeric_precision AS VARCHAR)) + ',' + RTRIM(CAST(numeric_scale AS VARCHAR)) + ')','(' + RTRIM(CAST(datetime_precision AS VARCHAR)) + ')','') + ', ' +
          CASE WHEN is_nullable = 'YES' THEN 'null' ELSE 'not null' END + ')' AS Columns
      FROM information_schema.columns c WHERE c.table_name='%s' AND c.TABLE_SCHEMA = '%s'`,

		opts.Table,
		opts.Schema,
	)

	foreignKeys := fmt.Sprintf(`
      SELECT c.constraint_name
         , kcu.column_name AS column_name
         , c2.table_name AS foreign_table_name
         , kcu2.column_name AS foreign_column_name
      FROM information_schema.table_constraints c
            INNER JOIN information_schema.key_column_usage kcu
              ON c.constraint_schema = kcu.constraint_schema
                AND c.constraint_name = kcu.constraint_name
            INNER JOIN information_schema.referential_constraints rc
              ON c.constraint_schema = rc.constraint_schema
                AND c.constraint_name = rc.constraint_name
            INNER JOIN information_schema.table_constraints c2
              ON rc.unique_constraint_schema = c2.constraint_schema
                AND rc.unique_constraint_name = c2.constraint_name
            INNER JOIN information_schema.key_column_usage kcu2
              ON c2.constraint_schema = kcu2.constraint_schema
                AND c2.constraint_name = kcu2.constraint_name
                AND kcu.ordinal_position = kcu2.ordinal_position
      WHERE c.constraint_type = 'FOREIGN KEY'
      AND c.TABLE_NAME = '%s' AND c.TABLE_SCHEMA = '%s'`,

		opts.Table,
		opts.Schema,
	)

	references := fmt.Sprintf(`
      SELECT kcu1.constraint_name AS constraint_name
          , kcu1.table_name AS foreign_table_name
          , kcu1.column_name AS foreign_column_name
          , kcu2.column_name AS column_name
      FROM information_schema.referential_constraints AS rc
      INNER JOIN information_schema.key_column_usage AS kcu1
          ON kcu1.constraint_catalog = rc.constraint_catalog
          AND kcu1.constraint_schema = rc.constraint_schema
          AND kcu1.constraint_name = rc.constraint_name
      INNER JOIN information_schema.key_column_usage AS kcu2
          ON kcu2.constraint_catalog = rc.unique_constraint_catalog
          AND kcu2.constraint_schema = rc.unique_constraint_schema
          AND kcu2.constraint_name = rc.unique_constraint_name
          AND kcu2.ordinal_position = kcu1.ordinal_position
      WHERE kcu2.table_name='%s' AND kcu2.table_schema = '%s'`,

		opts.Table,
		opts.Schema,
	)

	primaryKeys := fmt.Sprintf(`
       SELECT tc.constraint_name, kcu.column_name
       FROM
           information_schema.table_constraints AS tc
           JOIN information_schema.key_column_usage AS kcu
             ON tc.constraint_name = kcu.constraint_name
           JOIN information_schema.constraint_column_usage AS ccu
             ON ccu.constraint_name = tc.constraint_name
      WHERE constraint_type = 'PRIMARY KEY'
      AND tc.table_name = '%s' AND tc.table_schema = '%s'`,

		opts.Table,
		opts.Schema,
	)

	constraints := fmt.Sprintf(`
      SELECT u.CONSTRAINT_NAME, c.CHECK_CLAUSE FROM INFORMATION_SCHEMA.CONSTRAINT_TABLE_USAGE u
          INNER JOIN INFORMATION_SCHEMA.CHECK_CONSTRAINTS c ON u.CONSTRAINT_NAME = c.CONSTRAINT_NAME
      WHERE TABLE_NAME = '%s' AND u.TABLE_SCHEMA = '%s'`,

		opts.Table,
		opts.Schema,
	)

	return map[string]string{
		"List":         fmt.Sprintf("SELECT top 200 * from [%s]", opts.Table),
		"Columns":      columnSummary,
		"Indexes":      fmt.Sprintf("exec sp_helpindex '%s.%s'", opts.Schema, opts.Table),
		"Foreign Keys": foreignKeys,
		"References":   references,
		"Primary Keys": primaryKeys,
		"Constraints":  constraints,
		"Describe":     fmt.Sprintf("exec sp_help ''%s.%s''", opts.Schema, opts.Table),
	}
}
