package adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArango_Connect(t *testing.T) {
	tests := []struct {
		name          string
		connectionURL string
		wantErr       bool
		messageErr    string
		database      string
	}{
		{
			name:          "should fail with invalid url format",
			connectionURL: "://invalid",
			wantErr:       true,
			messageErr:    "failed to parse connection string",
		},
		{
			name:          "should fail with missing password",
			connectionURL: "http://token@hostname:8529",
			wantErr:       true,
			messageErr:    "arango: missing password",
		},
		{
			name:          "should fail with missing password with root user",
			connectionURL: "http://root@hostname.com:8529",
			wantErr:       true,
			messageErr:    "arango: missing password",
		},
		{
			name:          "should succeed with valid connection with root user",
			connectionURL: "http://root@hostname.com:8529?allow_empty_root_password=true",
			database:      "_system",
		},
		{
			name:          "should succeed with valid connection",
			connectionURL: "http://token:dummytoken@hostname.com:8529",
			database:      "_system",
		},
		{
			name:          "should succeed to parse database name from path",
			connectionURL: "http://token:dummytoken@hostname.com:8529/_db/testdb",
			database:      "testdb",
		},
		{
			name:          "should succeed with valid connection with options",
			connectionURL: "http://token:dummytoken@hostname.com:8529/_db/testdb?insecure_skip_verify",
			database:      "testdb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &Arango{}
			got, err := d.Connect(tt.connectionURL)

			if tt.wantErr {
				assert.NotEqual(t, "", tt.messageErr)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.messageErr)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.Equal(t, tt.database, got.(*arangoDriver).dbName)
		})
	}
}
