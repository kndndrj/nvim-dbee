package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trinodb/trino-go-client/trino"
)

// setupTrinoTestDriver helper function to setup trino driver for testing
func setupTrinoTestDriver(t *testing.T) (*trinoDriver, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	parsedURL, _ := url.Parse("http://localhost:8080?catalog=test_catalog")
	driver := &trinoDriver{
		client:        db,
		cfg:           &trino.Config{Catalog: "test_catalog"},
		connectionURL: parsedURL,
	}

	return driver, mock
}

func Test_trinoDriver_Query(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantRows *sqlmock.Rows
		wantErr  bool
	}{
		{
			name:  "simple select query",
			query: "SELECT * FROM test",
			wantRows: sqlmock.NewRows([]string{"col1", "col2"}).
				AddRow("value1", "value2"),
		},
		{
			name:    "invalid query",
			query:   "INVALID QUERY",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, mock := setupTrinoTestDriver(t)

			if tt.wantErr {
				mock.ExpectQuery(tt.query).WillReturnError(sql.ErrConnDone)
			} else {
				mock.ExpectQuery(tt.query).WillReturnRows(tt.wantRows)
			}

			got, err := driver.Query(context.Background(), tt.query)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func Test_trinoDriver_Columns(t *testing.T) {
	tests := []struct {
		name        string
		opts        *core.TableOptions
		mockRows    *sqlmock.Rows
		wantColumns []*core.Column
		wantErr     bool
	}{
		{
			name: "valid table columns",
			opts: &core.TableOptions{Schema: "test_schema", Table: "test_table"},
			mockRows: sqlmock.NewRows([]string{"column_name", "data_type"}).
				AddRow("id", "integer").
				AddRow("name", "varchar"),
			wantColumns: []*core.Column{
				{Name: "id", Type: "integer"},
				{Name: "name", Type: "varchar"},
			},
		},
		{
			name:    "invalid table",
			opts:    &core.TableOptions{Schema: "invalid", Table: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, mock := setupTrinoTestDriver(t)

			expectedQuery := fmt.Sprintf(`
        SELECT column_name, data_type
        FROM %s.information_schema.columns
        WHERE table_catalog = ? AND table_schema = ? AND table_name = ?
        ORDER BY ordinal_position`, trinoQuoteIdentifier("test_catalog"))

			if tt.wantErr {
				mock.ExpectQuery(expectedQuery).WillReturnError(sql.ErrConnDone)
			} else {
				mock.ExpectQuery(expectedQuery).
					WithArgs("test_catalog", tt.opts.Schema, tt.opts.Table).
					WillReturnRows(tt.mockRows)
			}

			got, err := driver.Columns(tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantColumns, got)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func Test_trinoDriver_Structure(t *testing.T) {
	tests := []struct {
		name          string
		catalogRows   *sqlmock.Rows
		schemaRows    *sqlmock.Rows
		tableRows     *sqlmock.Rows
		wantStructure []*core.Structure
		wantErr       bool
	}{
		{
			name: "valid structure",
			catalogRows: sqlmock.NewRows([]string{"catalog"}).
				AddRow("test_catalog"),
			schemaRows: sqlmock.NewRows([]string{"schema"}).
				AddRow("test_schema"),
			tableRows: sqlmock.NewRows([]string{"table_schema", "table_name", "table_type"}).
				AddRow("test_schema", "test_table", "TABLE").
				AddRow("test_schema", "test_view", "VIEW"),
			wantStructure: []*core.Structure{
				{
					Name: "test_catalog",
					Type: core.StructureTypeSchema,
					Children: []*core.Structure{
						{
							Name:   "test_schema",
							Schema: "test_catalog",
							Type:   core.StructureTypeSchema,
							Children: []*core.Structure{
								{Name: "test_table", Schema: "test_schema", Type: core.StructureTypeTable},
								{Name: "test_view", Schema: "test_schema", Type: core.StructureTypeView},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver, mock := setupTrinoTestDriver(t)

			mock.ExpectQuery("SHOW CATALOGS").WillReturnRows(tt.catalogRows)
			mock.ExpectQuery(`SHOW SCHEMAS FROM "test_catalog"`).WillReturnRows(tt.schemaRows)
			mock.ExpectQuery(`
                SELECT table_schema, table_name, table_type
                FROM "test_catalog".information_schema.tables
                WHERE table_schema = 'test_schema'
                AND table_schema NOT IN ('information_schema')`).WillReturnRows(tt.tableRows)

			got, err := driver.Structure()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantStructure, got)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
