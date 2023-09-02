package clients

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

func TestNewRedshift(t *testing.T) {
	type args struct {
		rawURL string
	}
	tests := []struct {
		want    *RedshiftClient
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid URL",
			args: args{
				rawURL: "postgres://user:password@localhost:5432/dbname?sslmode=disable",
			},
			want: &RedshiftClient{
				c: &mockClient{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRedshift(tt.args.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRedshift() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				require.Nil(t, got)
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRedshiftClient_Query(t *testing.T) {
	type fields struct {
		c common.DatabaseClient
	}
	type args struct {
		query string
	}
	tests := []struct {
		want    models.IterResult
		fields  fields
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid query",
			args: args{
				query: "SELECT * FROM table",
			},
			fields: fields{
				c: &mockClient{
					ConnFn: func() (common.DatabaseConnection, error) {
						return &mockConnection{
							QueryFn: func(query string) (models.IterResult, error) {
								return &mockIterResult{
									MetaFn: func() (models.Meta, error) {
										return models.Meta{
											Query: "SELECT * FROM table",
										}, nil
									},
									HeaderFn: func() (models.Header, error) {
										return models.Header{}, nil
									},
									NextFn: func() (models.Row, error) {
										return models.Row{
											"col1",
											"col2",
										}, nil
									},
								}, nil
							},
							CloseFn: func() error {
								return nil
							},
						}, nil
					},
				},
			},
			want: &mockIterResult{
				MetaFn: func() (models.Meta, error) {
					return models.Meta{
						Query: "SELECT * FROM table",
					}, nil
				},
				HeaderFn: func() (models.Header, error) {
					return models.Header{}, nil
				},
				NextFn: func() (models.Row, error) {
					return models.Row{
						"col1",
						"col2",
					}, nil
				},
			},
		},
		{
			name: "Invalid query",
			args: args{
				query: "SELECT * table",
			},
			fields: fields{
				c: &mockClient{
					ConnFn: func() (common.DatabaseConnection, error) {
						return &mockConnection{
							QueryFn: func(query string) (models.IterResult, error) {
								return nil, fmt.Errorf("invalid query")
							},
							CloseFn: func() error {
								return nil
							},
						}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid connection",
			fields: fields{
				c: &mockClient{
					ConnFn: func() (common.DatabaseConnection, error) {
						return nil, fmt.Errorf("invalid connection")
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RedshiftClient{
				c: tt.fields.c,
			}
			got, err := c.Query(tt.args.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedshiftClient.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				require.Nil(t, got)
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			wantMeta, _ := tt.want.Meta()
			gotMeta, _ := got.Meta()
			require.Equal(t, wantMeta.Query, gotMeta.Query)
			wantNext, _ := tt.want.Next()
			gotNext, _ := got.Next()
			require.Equal(t, wantNext, gotNext)
		})
	}
}

func TestRedshiftClient_Close(t *testing.T) {
	type fields struct {
		c common.DatabaseClient
	}
	tests := []struct {
		fields fields
		name   string
	}{
		{
			name: "Valid close",
			fields: fields{
				c: &mockClient{
					ConnFn: func() (common.DatabaseConnection, error) {
						return &mockConnection{
							CloseFn: func() error {
								return nil
							},
						}, nil
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RedshiftClient{
				c: tt.fields.c,
			}
			c.Close()
			require.NotNil(t, c)
		})
	}
}

func TestRedshiftClient_Layout(t *testing.T) {
	type fields struct {
		c common.DatabaseClient
	}
	tests := []struct {
		name    string
		fields  fields
		want    []models.Layout
		wantErr bool
	}{
		{
			name: "Valid layout",
			fields: fields{
				c: &mockClient{
					ConnFn: func() (common.DatabaseConnection, error) {
						return &mockConnection{
							QueryFn: func(query string) (models.IterResult, error) {
								return &mockIterResult{
									MetaFn: func() (models.Meta, error) {
										return models.Meta{
											Query: "SELECT * FROM table",
										}, nil
									},
									HeaderFn: func() (models.Header, error) {
										return models.Header{}, nil
									},
									NextFn: func() (models.Row, error) {
										// we expect results as
										// schema, table, type
										// for redshift.
										// e.g.
										// return models.Row{
										// 	"public", "mytable1", "TABLE",
										// 	"public", "mytable2", "TABLE",
										// 	"public", "mytable3", "TABLE",
										// 	"public", "myview1", "VIEW",
										// 	"public", "myview2", "VIEW",
										// 	"public", "myview3", "VIEW",
										// }, nil
										// but difficult to test this.
										return nil, nil
									},
								}, nil
							},
							CloseFn: func() error {
								return nil
							},
						}, nil
					},
				},
			},
			want: []models.Layout{},
		},
		{
			name: "Invalid row.Next",
			fields: fields{
				c: &mockClient{
					ConnFn: func() (common.DatabaseConnection, error) {
						return &mockConnection{
							QueryFn: func(query string) (models.IterResult, error) {
								return &mockIterResult{
									MetaFn: func() (models.Meta, error) {
										return models.Meta{
											Query: "SELECT * FROM table",
										}, nil
									},
									HeaderFn: func() (models.Header, error) {
										return models.Header{}, nil
									},
									NextFn: func() (models.Row, error) {
										return models.Row{}, fmt.Errorf("invalid next err")
									},
								}, nil
							},
							CloseFn: func() error {
								return nil
							},
						}, nil
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RedshiftClient{
				c: tt.fields.c,
			}
			got, err := c.Layout()
			if (err != nil) != tt.wantErr {
				t.Errorf("RedshiftClient.Layout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				require.Nil(t, got)
				require.Error(t, err)
				return
			}
			require.Len(t, got, 0)
		})
	}
}

func Test_fetchPsqlLayouts(t *testing.T) {
	// NOTE: this function currently tests partly the fetchPsqlLayouts
	// function. Mainly because NextFn needs to be refactored so it return
	// different models.Row upon each call during the while loop.
	// This should be tested in the future.
	type args struct {
		rows   models.IterResult
		dbType string
	}
	tests := []struct {
		name    string
		args    args
		want    []models.Layout
		wantErr bool
	}{
		{
			name: "Valid layout redshift",
			args: args{
				rows: &mockIterResult{
					NextFn: func() (models.Row, error) {
						return nil, nil
					},
				},
				dbType: "redshift",
			},
		},
		{
			name: "Valid layout postgres",
			args: args{
				rows: &mockIterResult{
					NextFn: func() (models.Row, error) {
						return nil, nil
					},
				},
				dbType: "postgres",
			},
		},
		{
			name: "Invalid layout",
			args: args{
				rows: &mockIterResult{
					NextFn: func() (models.Row, error) {
						return models.Row{}, fmt.Errorf("err next")
					},
				},
				dbType: "postgres",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchPGLayouts(tt.args.rows, tt.args.dbType)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchPsqlLayouts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				require.Nil(t, got)
				require.Error(t, err)
				return
			}
			require.Len(t, got, 0)
		})
	}
}

func Test_getLayoutType(t *testing.T) {
	type args struct {
		typ string
	}
	tests := []struct {
		name string
		args args
		want models.LayoutType
	}{
		{
			name: "Table type",
			args: args{
				typ: "TABLE",
			},
			want: models.LayoutTypeTable,
		},
		{
			name: "View type",
			args: args{
				typ: "VIEW",
			},
			want: models.LayoutTypeView,
		},
		{
			name: "Default type",
			args: args{
				typ: "",
			},
			want: models.LayoutTypeNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getLayoutType(tt.args.typ)
			require.Equal(t, tt.want, got)
		})
	}
}

type mockClient struct {
	ConnFn func() (common.DatabaseConnection, error)
}

func (m *mockClient) Conn() (common.DatabaseConnection, error) {
	return m.ConnFn()
}

func (m *mockClient) Close() {}

func (m *mockClient) Swap(db *sql.DB) {}

type mockConnection struct {
	CloseFn func() error
	ExecFn  func(query string) (models.IterResult, error)
	QueryFn func(query string) (models.IterResult, error)
}

func (m *mockConnection) Close() error {
	return m.CloseFn()
}

func (m *mockConnection) Exec(query string) (models.IterResult, error) {
	return m.ExecFn(query)
}

func (m *mockConnection) Query(query string) (models.IterResult, error) {
	return m.QueryFn(query)
}

type mockIterResult struct {
	MetaFn   func() (models.Meta, error)
	HeaderFn func() (models.Header, error)
	NextFn   func() (models.Row, error)
	ErrFn    func() error
}

func (m *mockIterResult) Meta() (models.Meta, error) {
	return m.MetaFn()
}

func (m *mockIterResult) Header() (models.Header, error) {
	return m.HeaderFn()
}

func (m *mockIterResult) Next() (models.Row, error) {
	return m.NextFn()
}

func (m *mockIterResult) Close() {}

func (m *mockIterResult) Err() error {
	return m.ErrFn()
}

func (m *mockIterResult) SetCustomHeader(h models.Header) {}

func (m *mockIterResult) SetCallback(cb func()) {}
