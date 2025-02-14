package adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseDatabaseFromPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "should return `test` part from unix file path",
			input: "/tmp/test.db",
			want:  "test",
		},
		{
			name:  "should return `.hiddenFile` part from unix file path",
			input: "/tmp/.hiddenFile.db",
			want:  "hiddenFile",
		},
		{
			name:  "should return `my_file` part from file url path",
			input: "file:///tmp/my_file.database",
			want:  "my_file",
		},
		{
			name:  "should return `my_db` part from s3 bucket url",
			input: "s3://bucket_name/path/to/my_db.duckdb",
			want:  "my_db",
		},
		{
			name:  "should return `remote_db` part from https url",
			input: "https://www.example.com/remote_db.example.new",
			want:  "remote_db",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDatabaseFromPath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
