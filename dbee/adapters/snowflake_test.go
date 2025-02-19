package adapters

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

func TestNewSnowflake(t *testing.T) {
	r := require.New(t)

	type args struct {
		rawURL string
	}
	tests := []struct {
		want    *snowflakeDriver
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid URL",
			args: args{
				rawURL: "snowflake://user:password@my_organization-my_account/mydb",
			},
			want: &snowflakeDriver{
				c: &builders.Client{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := new(Snowflake).Connect(tt.args.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSnowflake() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				r.Nil(got)
				r.Error(err)
				return
			}
			r.NoError(err)
		})
	}
}
