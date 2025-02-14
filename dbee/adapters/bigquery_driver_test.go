package adapters

import (
	"net/url"
	"reflect"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/stretchr/testify/assert"
)

func Test_toKebabCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "should convert to kebab case single",
			input: "helloWorld",
			want:  "hello-world",
		},
		{
			name:  "should convert to kebab case multiple",
			input: "fooBasYes",
			want:  "foo-bas-yes",
		},
		{
			name:  "should convert to kebab case with many upper case",
			input: "helloSQL",
			want:  "hello-sql",
		},
		{
			name:  "should not convert kebab case with upper case beginning",
			input: "Hello",
			want:  "hello",
		},
		{
			name:  "should not convert kebab case with upper case at the end",
			input: "hellO",
			want:  "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toKebabCase(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_bigQueryDriver_buildHeader(t *testing.T) {
	type args struct {
		parentName string
		schema     bigquery.Schema
	}
	tests := []struct {
		name string
		args args
		want core.Header
	}{
		{
			name: "should build header with no nested fields",
			args: args{parentName: "", schema: bigquery.Schema{
				{
					Name: "foo1",
					Type: bigquery.StringFieldType,
				},
				{
					Name: "foo2",
					Type: bigquery.StringFieldType,
				},
			}},
			want: []string{"foo1", "foo2"},
		},
		{
			name: "should build header with nested fields but no parent",
			args: args{schema: bigquery.Schema{
				{
					Name: "foo",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{
							Name: "nested_foo",
							Type: bigquery.StringFieldType,
						},
					},
				},
			}},
			want: []string{"nested_foo"},
		},
		{
			name: "should build header with nested fields and parent",
			args: args{parentName: "parent_foo", schema: bigquery.Schema{
				{
					Name: "foo",
					Type: bigquery.RecordFieldType,
					Schema: bigquery.Schema{
						{
							Name: "nested_foo",
							Type: bigquery.StringFieldType,
						},
					},
				},
			}},
			want: []string{"nested_foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &bigQueryDriver{
				c:           &bigquery.Client{},
				QueryConfig: bigquery.QueryConfig{},
			}
			got := d.buildHeader(tt.args.parentName, tt.args.schema)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_setQueryConfigFromParams(t *testing.T) {
	type args struct {
		cfg    *bigquery.QueryConfig
		params url.Values
	}
	tests := []struct {
		name    string
		args    args
		want    *bigquery.QueryConfig
		wantErr bool
	}{
		{
			name: "should set field when it exists and correct data type",
			args: args{
				cfg: &bigquery.QueryConfig{},
				params: url.Values{
					"max-bytes-billed": []string{"10"},
				},
			},
			want: &bigquery.QueryConfig{MaxBytesBilled: 10},
		},
		{
			name: "should not set when field does not exist",
			args: args{
				cfg:    &bigquery.QueryConfig{},
				params: url.Values{"does not exist": []string{}},
			},
			want: &bigquery.QueryConfig{},
		},
		{
			name: "should not set when data type isn't supported",
			args: args{
				cfg:    &bigquery.QueryConfig{},
				params: url.Values{"table-definitions": []string{}},
			},
			want: &bigquery.QueryConfig{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setQueryConfigFromParams(tt.args.cfg, tt.args.params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, tt.args.cfg)
		})
	}
}

func Test_setFieldFromString(t *testing.T) {
	type args struct {
		fieldName  string
		fieldValue string
	}
	tests := []struct {
		name    string
		args    args
		want    *bigquery.QueryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "should set string field",
			args: args{
				fieldName:  "DefaultProjectID",
				fieldValue: "foo",
			},
			want: &bigquery.QueryConfig{DefaultProjectID: "foo"},
		},
		{
			name: "should set bool field",
			args: args{
				fieldName:  "DisableQueryCache",
				fieldValue: "false",
			},
			want: &bigquery.QueryConfig{DisableQueryCache: false},
		},
		{
			name: "should set int64 field",
			args: args{
				fieldName:  "MaxBytesBilled",
				fieldValue: "10",
			},
			want: &bigquery.QueryConfig{MaxBytesBilled: int64(10)},
		},
		{
			name: "should set int field",
			args: args{
				fieldName:  "MaxBillingTier",
				fieldValue: "10",
			},
			want: &bigquery.QueryConfig{MaxBillingTier: 10},
		},
		{
			name: "should set time.Duration field (alias for int64)",
			args: args{
				fieldName:  "JobTimeout",
				fieldValue: "10000",
			},
			want: &bigquery.QueryConfig{JobTimeout: 10000},
		},
		{
			name: "should set WriteDisposition field (alias for string)",
			args: args{
				fieldName:  "WriteDisposition",
				fieldValue: "WRITE_TRUNCATE",
			},
			want: &bigquery.QueryConfig{WriteDisposition: "WRITE_TRUNCATE"},
		},
		{
			name: "should return nil when val is nil",
			args: args{
				fieldName:  "WriteDisposition",
				fieldValue: "",
			},
			want: &bigquery.QueryConfig{},
		},
		{
			name: "should error when field is not settable",
			args: args{
				fieldName:  "forceStorageAPI", // unexported field
				fieldValue: "true",
			},
			wantErr: true,
			errMsg:  "field is not settable",
		},
		{
			name: "should error when field type unsupported",
			args: args{
				fieldName:  "TableDefinitions",
				fieldValue: "{'hello': 'world'}",
			},
			wantErr: true,
			errMsg:  "unsupported field type map",
		},
		{
			name: "should error when unable to parse bool",
			args: args{
				fieldName:  "DisableQueryCache",
				fieldValue: "not a bool",
			},
			wantErr: true,
			errMsg:  "failed to parse bool",
		},
		{
			name: "should error when unable to parse int",
			args: args{
				fieldName:  "MaxBytesBilled",
				fieldValue: "not a number",
			},
			wantErr: true,
			errMsg:  "failed to parse int",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &bigquery.QueryConfig{}
			v := reflect.ValueOf(cfg).Elem()

			err := setFieldFromString(v.FieldByName(tt.args.fieldName), tt.args.fieldValue)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			assert.NoError(t, err)
			assert.EqualValues(t, tt.want, cfg)
		})
	}
}
