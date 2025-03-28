package adapters_test

import (
	"log"
	"testing"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
)

func TestArango_Connect(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		rawUrl  string
		want    core.Driver
		wantErr bool
	}{
		struct {
			name    string
			rawUrl  string
			want    core.Driver
			wantErr bool
		}{
			name:    "happy path",
			rawUrl:  "http://root:rootpassword@localhost:8529",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "bad url",
			rawUrl:  "root:rootpassword@:8529",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p adapters.Arango
			got, gotErr := p.Connect(tt.rawUrl)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Connect() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Connect() succeeded unexpectedly")
			}
			cleanup := func() {
				got.Close()
			}
			if true {
				structure, err := got.Structure()
				if err != nil {
					log.Fatalf("Error: %v", err)
					t.Errorf("Connect() failed: %v", err)
				}
				for i := range structure {
					log.Printf("structure: %v", structure[i].Name)
					for j := range structure[i].Children {
						log.Printf("\tchild: %v", structure[i].Children[j].Name)
					}
				}

				t.Cleanup(cleanup)
			}
		})
	}
}
