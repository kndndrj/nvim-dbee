package adapters

import (
	"testing"

	_ "github.com/databricks/databricks-sql-go"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabricks_Connect(t *testing.T) {
	tests := []struct {
		name          string
		connectionURL string
		wantErr       bool
		messageErr    string
	}{
		{
			name:          "should fail with invalid url format",
			connectionURL: "://invalid",
			wantErr:       true,
			messageErr:    "failed to parse connection string",
		},
		{
			name:          "should fail with missing catalog",
			connectionURL: "token:dummytoken@hostname:443/sql/1.0/endpoints/1234567890",
			wantErr:       true,
			messageErr:    "required parameter '?catalog=<catalog>' is missing",
		},
		{
			name:          "should succeed with valid connection",
			connectionURL: "token:dummytoken@hostname:443/sql/1.0/endpoints/1234567890?catalog=my_catalog",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &Databricks{}
			got, err := d.Connect(tt.connectionURL)

			if tt.wantErr {
				assert.NotEqual(t, "", tt.messageErr)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.messageErr)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestDatabricks_GetHelpers(t *testing.T) {
	defaultOpts := &core.TableOptions{
		Schema:          "test_schema",
		Table:           "test_table",
		Materialization: core.StructureTypeTable,
	}
	tests := []struct {
		name string
		key  string
		opts *core.TableOptions
		want string
	}{
		{
			name: "should return list query",
			key:  "List",
			opts: defaultOpts,
			want: "SELECT * FROM test_schema.test_table LIMIT 100;",
		},
		{
			name: "should return columns query",
			key:  "Columns",
			opts: defaultOpts,
			want: "\n\t\tSELECT *\n\t\tFROM information_schema.column\n\t\tWHERE table_schema = 'test_schema'\n\t\t\tAND table_name = 'test_table';",
		},
		{
			name: "should return describe query",
			key:  "Describe",
			opts: defaultOpts,
			want: "DESCRIBE EXTENDED test_schema.test_table;",
		},
		{
			name: "should return constraints query",
			key:  "Constraints",
			opts: defaultOpts,
			want: "\n\t\tSELECT *\n\t\tFROM information_schema.table_constraints\n\t\tWHERE table_schema = 'test_schema'\n\t\t\tAND table_name = 'test_table';",
		},
		{
			name: "should return key_column_usage query",
			key:  "Keys",
			opts: defaultOpts,
			want: "\n\t\tSELECT *\n\t\tFROM information_schema.key_column_usage\n\t\tWHERE table_schema = 'test_schema'\n\t\t\tAND table_name = 'test_table';",
		},
	}

	d := &Databricks{}
	helpers := d.GetHelpers(defaultOpts)

	for helperKey := range helpers {
		var found bool
		for _, tt := range tests {
			if tt.key == helperKey {
				found = true
				break
			}
		}
		require.True(t, found, "missing test case for helper key: %q", helperKey)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := helpers[tt.key]
			assert.Equal(t, tt.want, got)
		})
	}
}
