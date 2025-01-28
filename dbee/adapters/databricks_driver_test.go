package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDatabricksTestDriver helper function to setup databricks driver for testing
func setupDatabricksTestDriver(t *testing.T) (*databricksDriver, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	driver := &databricksDriver{
		c:              builders.NewClient(db),
		connectionURL:  &url.URL{},
		currentCatalog: "test_catalog",
	}

	return driver, mock
}

func Test_databricksDriver_Query(t *testing.T) {
	tests := []struct {
		give     string
		wantRows *sqlmock.Rows
		wantErr  bool
	}{
		{
			give: "SELECT * FROM test",
			wantRows: sqlmock.NewRows([]string{"col1", "col2"}).
				AddRow("value1", "value2"),
		},
		{
			give:    "INVALID QUERY",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.give, func(t *testing.T) {
			t.Parallel()

			driver, mock := setupDatabricksTestDriver(t)

			if tt.wantErr {
				mock.ExpectQuery(tt.give).WillReturnError(sql.ErrConnDone)
			} else {
				mock.ExpectQuery(tt.give).WillReturnRows(tt.wantRows)
			}

			got, err := driver.Query(context.Background(), tt.give)

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

func Test_databricksDriver_Columns(t *testing.T) {
	tests := []struct {
		name        string
		give        *core.TableOptions
		input       *sqlmock.Rows
		wantColumns []*core.Column
		wantErr     bool
	}{
		{
			name: "should succeed with cols found",
			give: &core.TableOptions{Schema: "public", Table: "users"},
			input: sqlmock.NewRows([]string{"column_name", "data_type"}).
				AddRow("id", "integer").
				AddRow("name", "varchar"),
			wantColumns: []*core.Column{
				{Name: "id", Type: "integer"},
				{Name: "name", Type: "varchar"},
			},
		},
		{
			name: "should fail with not found",
			give: &core.TableOptions{
				Schema: "invalid",
				Table:  "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			driver, mock := setupDatabricksTestDriver(t)
			expectedQuery := fmt.Sprintf(`
                SELECT column_name, data_type
                FROM information_schema.columns
                WHERE
                    table_schema='%s' AND
                    table_name='%s';`,
				tt.give.Schema, tt.give.Table)

			if tt.wantErr {
				mock.ExpectQuery(expectedQuery).WillReturnError(sql.ErrConnDone)
			} else {
				mock.ExpectQuery(expectedQuery).WillReturnRows(tt.input)
			}

			got, err := driver.Columns(tt.give)

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

func Test_databricksDriver_Structure(t *testing.T) {
	tests := []struct {
		name     string
		testRows *sqlmock.Rows
		want     []*core.Structure
		wantErr  bool
	}{
		{
			name: "should succeed with tables and views",
			testRows: sqlmock.NewRows([]string{"table_schema", "table_name", "table_type"}).
				AddRow("public", "users", "TABLE").
				AddRow("public", "user_view", "VIEW"),
			want: []*core.Structure{
				{Name: "users", Schema: "public", Type: core.StructureTypeTable},
				{Name: "user_view", Schema: "public", Type: core.StructureTypeView},
			},
		},
		{
			name:    "should fail with error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			driver, mock := setupDatabricksTestDriver(t)
			expectedQuery := `
                SELECT table_schema, table_name, table_type
                FROM system.information_schema.tables
                WHERE table_catalog = 'test_catalog'; `

			if tt.wantErr {
				mock.ExpectQuery(expectedQuery).WillReturnError(sql.ErrConnDone)
			} else {
				mock.ExpectQuery(expectedQuery).WillReturnRows(tt.testRows)
			}

			got, err := driver.Structure()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			for _, g := range got {
				assert.Equal(t, core.StructureTypeSchema, g.Type)
				assert.Equal(t, tt.want, g.Children)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func Test_getDatabricksStructureType(t *testing.T) {
	tests := []struct {
		name string
		give string
		want core.StructureType
	}{
		{
			name: "should return table with table",
			give: "TABLE",
			want: core.StructureTypeTable,
		},
		{
			name: "should return table with system table",
			give: "SYSTEM TABLE",
			want: core.StructureTypeTable,
		},
		{
			name: "should return view with view",
			give: "VIEW",
			want: core.StructureTypeView,
		},
		{
			name: "should return none with unknown",
			give: "UNKNOWN",
			want: core.StructureTypeNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getDatabricksStructureType(tt.give)
			assert.Equal(t, tt.want, got)
		})
	}
}
