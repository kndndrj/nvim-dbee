package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"sort"
	"testing"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"
	"github.com/google/uuid"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

func setupArangoDBTest(t *testing.T) (*arangoDriver, context.Context, *arangodb.Collection) {
	// Set up ArangoDB test server
	rawUrl := "http://root:rootpassword@localhost:8529"
	ctx := context.Background()
	u, err := url.Parse(rawUrl)
	if err != nil {
		t.Fatal(err)
	}

	// Set authentication
	dbUrl := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	endpoint := connection.NewRoundRobinEndpoints([]string{dbUrl})
	conn := connection.NewHttpConnection(connection.DefaultHTTPConfigurationWrapper(endpoint, true))

	var auth connection.Wrapper
	if u.User != nil {
		// Basic Authentication
		username := u.User.Username()
		password, _ := u.User.Password()
		auth = connection.NewJWTAuthWrapper(username, password)
		conn = auth(conn)
	}

	conn = connection.NewConnectionAsyncWrapper(conn)

	// Create a client
	client := arangodb.NewClient(conn)
	if err != nil {
		t.Fatal(err)
	}

	// Create a test database and collection
	dbName := fmt.Sprintf("testdb-%s", uuid.New().String())
	db, err := client.CreateDatabase(ctx, dbName, &arangodb.CreateDatabaseOptions{})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db, err := client.GetDatabase(ctx, dbName, nil)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		err = db.Remove(ctx)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	})

	collection, err := db.CreateCollection(ctx, "testcollection", &arangodb.CreateCollectionProperties{})
	if err != nil {
		t.Fatal(err)
	}

	// Create some sample documents
	doc1 := map[string]interface{}{
		"name": "John",
		"age":  30,
	}
	doc2 := map[string]interface{}{
		"name": "Jane",
		"age":  25,
	}

	doc3 := map[string]interface{}{
		"name": "Joe",
	}
	_, err = collection.CreateDocument(ctx, doc1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = collection.CreateDocument(ctx, doc2)
	if err != nil {
		t.Fatal(err)
	}
	_, err = collection.CreateDocument(ctx, doc3)
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("collection: %v", collection)

	// Initialize the arangoDriver struct
	a := &arangoDriver{
		c:      &client,
		dbName: dbName,
	}

	return a, ctx, &collection
}

func Test_arangoDriver_Columns(t *testing.T) {
	a, _, _ := setupArangoDBTest(t)
	tests := []struct {
		name string
		// Named input parameters for target function.
		opts    *core.TableOptions
		want    []*core.Column
		wantErr bool
	}{
		{
			name: "get columns",
			opts: &core.TableOptions{
				Table: "testcollection",
			},
			want: []*core.Column{
				{
					Name: "name",
					Type: "collection",
				},
				{
					Name: "age",
					Type: "collection",
				},
				{
					Name: "_key",
					Type: "collection",
				},

				{
					Name: "_rev",
					Type: "collection",
				},
				{
					Name: "_id",
					Type: "collection",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := a.Columns(tt.opts)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Columns() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("Columns() succeeded unexpectedly")
			}
			if len(got) != len(tt.want) {
				t.Errorf("Columns() = %v, want %v", got, tt.want)
			}
			gotNames := make([]string, len(got))
			wantNames := make([]string, len(tt.want))
			for i := range got {
				gotNames[i] = got[i].Name
			}
			for i := range tt.want {
				wantNames[i] = tt.want[i].Name
			}
			sort.Strings(gotNames)
			sort.Strings(wantNames)
			if !reflect.DeepEqual(gotNames, wantNames) {
				t.Errorf("Columns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_arangoDriver_Query(t *testing.T) {
	a, _, _ := setupArangoDBTest(t)
	type opts struct {
		Context context.Context
		Query   string
	}
	tests := []struct {
		name string
		// Named input parameters for target function.
		opts    *opts
		want    []*core.Column
		wantErr bool
	}{
		{
			name: "get columns",
			opts: &opts{
				Context: context.Background(),
				Query:   "for n in testcollection limit 1000 return n",
			},
			want:    []*core.Column{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		fmt.Printf("tt.name: %v\n", tt.name)
		result, err := a.Query(tt.opts.Context, tt.opts.Query)
		if err != nil {
			t.Fatal(err)
		}
		for {
			if !result.HasNext() {
				break
			}

			data, err := result.Next()
			if err != nil {
				t.Fatal(err)
			}

			jsonData, err := json.MarshalIndent(&data, "", "\t")
			if err != nil {
				t.Fatal(err)
			}
			t.Fatalf("result: %s\n", jsonData)
		}

		t.Fatal("result")

	}
}
