package adapters

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

func TestNewRedshift(t *testing.T) {
	type args struct {
		rawURL string
	}
	tests := []struct {
		want    *redshiftDriver
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid URL",
			args: args{
				rawURL: "postgres://user:password@localhost:5432/dbname?sslmode=disable",
			},
			want: &redshiftDriver{
				c: &builders.Client{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := new(Redshift).Connect(tt.args.rawURL)
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
