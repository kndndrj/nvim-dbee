package adapters

import (
	"encoding/gob"

	"github.com/redis/go-redis/v9"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// Register client
func init() {
	_ = register(&Redis{}, "redis")

	// register known types with gob
	gob.Register(&redisResponse{})
	gob.Register([]any{})
	gob.Register(map[any]any{})
}

var _ core.Adapter = (*Redis)(nil)

type Redis struct{}

func (r *Redis) Connect(url string) (core.Driver, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     url,
		Password: "",
		DB:       0,
	})

	return &redisDriver{
		redis: c,
	}, nil
}

func (*Redis) GetHelpers(opts *core.HelperOptions) map[string]string {
	return map[string]string{
		"List": "KEYS *",
	}
}
