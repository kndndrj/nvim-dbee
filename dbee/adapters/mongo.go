package adapters

import (
	"context"
	"encoding/gob"
	"fmt"
	"net/url"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// Register client
func init() {
	_ = register(&Mongo{}, "mongo", "mongodb")

	// register known types with gob
	// full list available in go.mongodb.org/.../bson godoc
	gob.Register(&mongoResponse{})
	gob.Register(bson.A{})
	gob.Register(bson.M{})
	gob.Register(bson.D{})
	gob.Register(primitive.ObjectID{})
	// gob.Register(primitive.DateTime)
	gob.Register(primitive.Binary{})
	gob.Register(primitive.Regex{})
	// gob.Register(primitive.JavaScript)
	gob.Register(primitive.CodeWithScope{})
	gob.Register(primitive.Timestamp{})
	gob.Register(primitive.Decimal128{})
	// gob.Register(primitive.MinKey{})
	// gob.Register(primitive.MaxKey{})
	// gob.Register(primitive.Undefined{})
	gob.Register(primitive.DBPointer{})
	// gob.Register(primitive.Symbol)
}

var _ core.Adapter = (*Mongo)(nil)

type Mongo struct{}

func (m *Mongo) Connect(rawURL string) (core.Driver, error) {
	// get database name from url
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("mongo: invalid url: %w", err)
	}

	opts := options.Client().ApplyURI(rawURL)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, err
	}

	return &mongoDriver{
		c:      client,
		dbName: u.Path[1:],
	}, nil
}

func (*Mongo) GetHelpers(opts *core.TableOptions) map[string]string {
	return map[string]string{
		"List": fmt.Sprintf(`{"find": %q}`, opts.Table),
	}
}
