package adapters

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/arangodb/go-driver/v2/arangodb"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var (
	_ core.Driver           = (*arangoDriver)(nil)
	_ core.DatabaseSwitcher = (*arangoDriver)(nil)
)

type arangoDriver struct {
	c      arangodb.Client
	dbName string
}

func (a *arangoDriver) ListDatabases() (current string, available []string, err error) {
	databases, err := a.c.AccessibleDatabases(context.Background())
	if err != nil {
		return "", nil, fmt.Errorf("failed to list databases: %w", err)
	}

	databaseNames := make([]string, len(databases))
	for i, db := range databases {
		databaseNames[i] = db.Name()
	}

	return a.dbName, databaseNames, nil
}

func (a *arangoDriver) SelectDatabase(name string) error {
	a.dbName = name
	return nil
}

func (a *arangoDriver) Close() {}

func (a *arangoDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	db, err := a.c.GetDatabase(context.Background(), a.dbName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	aql := `
	FOR n IN @@col
		LIMIT 1000
		FOR a IN ATTRIBUTES(n)
		COLLECT attribute = a WITH COUNT INTO len
		SORT len DESC
		LIMIT 10
		SORT attribute
		RETURN {attribute}`

	bindVars := map[string]any{"@col": opts.Table}
	cursor, err := db.Query(context.Background(), aql, &arangodb.QueryOptions{BindVars: bindVars})
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer cursor.Close()

	columns := make([]*core.Column, 0)
	doc := make(map[string]any)

	for cursor.HasMore() {
		if _, err := cursor.ReadDocument(context.Background(), &doc); err != nil {
			return nil, fmt.Errorf("failed to read document: %w", err)
		}
		column, ok := doc["attribute"].(string)
		if !ok {
			return nil, fmt.Errorf("failed to read document: %w", err)
		}
		columns = append(columns, &core.Column{Type: "collection", Name: column})
	}

	return columns, nil
}

func (a *arangoDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	db, err := a.c.GetDatabase(ctx, a.dbName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	cursor, err := db.Query(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer cursor.Close()

	next, hasNext := builders.NextYield(func(yield func(...any)) error {
		for cursor.HasMore() {
			var doc any

			if _, err := cursor.ReadDocument(ctx, &doc); err != nil {
				return err
			}

			yield(NewArangoResponse(doc))
		}

		return nil
	})

	return builders.NewResultStreamBuilder().
		WithNextFunc(next, hasNext).
		WithHeader(core.Header{"Results"}).
		WithMeta(&core.Meta{SchemaType: core.SchemaLess}).
		Build(), nil
}

func (a *arangoDriver) Structure() ([]*core.Structure, error) {
	ctx := context.Background()
	databases := []arangodb.Database{}
	err := errors.New("database not selected")
	if a.dbName == "_system" { // if the database is _system, we'll walk all databases
		databases, err = a.c.Databases(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list databases: %w", err)
		}
	} else {
		database, err := a.c.GetDatabase(ctx, a.dbName, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get database: %w", err)
		}
		databases = []arangodb.Database{database}
	}

	structures := make([]*core.Structure, len(databases))
	for i, db := range databases {
		collections, err := db.Collections(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list collections: %w", err)
		}

		structures[i] = &core.Structure{
			Name:     db.Name(),
			Schema:   db.Name(),
			Type:     core.StructureTypeSchema,
			Children: make([]*core.Structure, len(collections)),
		}

		for j, collection := range collections {
			structures[i].Children[j] = &core.Structure{
				Name:   collection.Name(),
				Schema: db.Name(),
				Type:   core.StructureTypeTable,
			}
		}
	}
	return structures, nil
}

type arangoResponse struct {
	Value any
}

func NewArangoResponse(val any) any {
	return &arangoResponse{Value: val}
}

func (ar *arangoResponse) String() string {
	parsed, err := json.MarshalIndent(ar.Value, "", "  ")
	if err != nil {
		return fmt.Sprint(ar.Value)
	}
	return string(parsed)
}

func (ar *arangoResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(ar.Value)
}

func (ar *arangoResponse) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	if err := encoder.Encode(ar.Value); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (ar *arangoResponse) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	return decoder.Decode(&ar.Value)
}
