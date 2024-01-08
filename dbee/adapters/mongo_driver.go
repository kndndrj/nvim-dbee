package adapters

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.Driver           = (*mongoDriver)(nil)
	_ core.DatabaseSwitcher = (*mongoDriver)(nil)
)

type mongoDriver struct {
	c      *mongo.Client
	dbName string
}

func (c *mongoDriver) getCurrentDatabase(ctx context.Context) (string, error) {
	if c.dbName != "" {
		return c.dbName, nil
	}

	dbs, err := c.c.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return "", fmt.Errorf("failed to select default database: %w", err)
	}
	if len(dbs) < 1 {
		return "", fmt.Errorf("no databases found")
	}
	c.dbName = dbs[0]

	return c.dbName, nil
}

func (c *mongoDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return []*core.Column{
		{
			Name: "",
			Type: "collection",
		},
	}, nil
}

func (c *mongoDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	dbName, err := c.getCurrentDatabase(ctx)
	if err != nil {
		return nil, err
	}
	db := c.c.Database(dbName)

	var command any
	err = bson.UnmarshalExtJSON([]byte(query), false, &command)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal command: \"%v\" to bson: %v", query, err)
	}

	var resp bson.M
	err = db.RunCommand(ctx, command).Decode(&resp)
	if err != nil {
		return nil, err
	}

	// check if "cursor" field exists and create an appropriate func
	var next func() (core.Row, error)
	var hasNext func() bool

	cur, ok := resp["cursor"]
	if ok {
		next, hasNext = builders.NextYield(func(yield func(...any)) error {
			cursor := cur.(bson.M)
			if !ok {
				return errors.New("type assertion for cursor object failed")
			}

			for _, b := range cursor {
				batch, ok := b.(bson.A)
				if !ok {
					continue
				}
				for _, item := range batch {
					yield(newMongoResponse(item))
				}
			}
			return nil
		})
	} else {
		next, hasNext = builders.NextSingle(newMongoResponse(resp))
	}

	// build result
	result := builders.NewResultStreamBuilder().
		WithNextFunc(next, hasNext).
		WithHeader(core.Header{"Reply"}).
		WithMeta(&core.Meta{
			SchemaType: core.SchemaLess,
		}).
		Build()

	return result, nil
}

func (c *mongoDriver) Structure() ([]*core.Structure, error) {
	ctx := context.Background()

	dbName, err := c.getCurrentDatabase(ctx)
	if err != nil {
		return nil, err
	}

	collections, err := c.c.Database(dbName).ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}

	var structure []*core.Structure

	for _, coll := range collections {
		structure = append(structure, &core.Structure{
			Name:   coll,
			Schema: "",
			Type:   core.StructureTypeTable,
		})
	}

	return structure, nil
}

func (c *mongoDriver) Close() {
	_ = c.c.Disconnect(context.TODO())
}

func (c *mongoDriver) ListDatabases() (current string, available []string, err error) {
	ctx := context.Background()

	dbName, err := c.getCurrentDatabase(ctx)
	if err != nil {
		return "", nil, err
	}

	all, err := c.c.ListDatabaseNames(ctx, bson.D{{
		Key: "name",
		Value: bson.D{{
			Key: "$not",
			Value: bson.D{{
				Key:   "$regex",
				Value: dbName,
			}},
		}},
	}})
	if err != nil {
		return "", nil, fmt.Errorf("failed to retrieve database names: %w", err)
	}

	return dbName, all, nil
}

func (c *mongoDriver) SelectDatabase(name string) error {
	c.dbName = name
	return nil
}

// mongoResponse serves as a wrapper around the mongo response
// to stringify the return values
type mongoResponse struct {
	value any
}

func newMongoResponse(val any) *mongoResponse {
	return &mongoResponse{
		value: val,
	}
}

func (mr *mongoResponse) String() string {
	parsed, err := json.MarshalIndent(mr.value, "", "  ")
	if err != nil {
		return fmt.Sprint(mr.value)
	}
	return string(parsed)
}

func (mr *mongoResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(mr.value)
}

func (mr *mongoResponse) GobEncode() ([]byte, error) {
	var err error
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err = encoder.Encode(mr.value)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), err
}

func (mr *mongoResponse) GobDecode(buf []byte) error {
	var err error
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	err = decoder.Decode(&mr.value)
	if err != nil {
		return err
	}
	return err
}
