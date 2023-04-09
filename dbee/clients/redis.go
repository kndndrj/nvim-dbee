package clients

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	redis *redis.Client
}

func NewRedis(url string) (*RedisClient, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     url,
		Password: "",
		DB:       0,
	})

	return &RedisClient{
		redis: c,
	}, nil
}

func (c *RedisClient) Query(query string) (conn.IterResult, error) {

	q := strings.Split(query, " ")

	args := make([]any, len(q))
	for i, v := range q {
		args[i] = v
	}

	resp, err := c.redis.Do(context.Background(), args...).Result()
	if err != nil {
		return nil, err
	}

	// parse response
	var rows []conn.Row
	switch rpl := resp.(type) {
	case int64:
		rows = []conn.Row{{rpl}}
	case string:
		rows = []conn.Row{{rpl}}
	case []any:
		rows = sliceToRows(rpl, -1)
	case map[any]any:
		for k, v := range rpl {
			rows = append(rows, conn.Row{k, v})
		}
	case nil:
		return nil, errors.New("no reponse from redis")
	default:
		return nil, fmt.Errorf("unknown type reponse from redis: %T!\n", rpl)
	}

	// build result
	max := len(rows) - 1
	i := 0
	result := common.NewResultBuilder().
		WithNextFunc(func() (conn.Row, error) {
			if i > max {
				return nil, nil
			}
			val := rows[i]
			i++
			return val, nil
		}).
		WithHeader(conn.Header{"Reply"}).
		WithMeta(conn.Meta{
			Query:     query,
			Timestamp: time.Now(),
		}).
		Build()

	return result, err
}

func (c *RedisClient) Schema() (conn.Schema, error) {
	return conn.Schema{
		"DB": []string{"DB"},
	}, nil
}

func (c *RedisClient) Close() {
	c.redis.Close()
}

// sliceToRows expands []any slice and any possible nested slices to multiple rows
func sliceToRows(slice []any, level int) []conn.Row {
	var rows []conn.Row

	var prefix []any
	for i := 0; i < level; i++ {
		prefix = append(prefix, "")
	}

	for _, v := range slice {
		if nested, ok := v.([]any); ok {
			rs := sliceToRows(nested, level+1)
			rows = append(rows, rs...)
		} else {
			row := append(prefix, v)
			rows = append(rows, row)
		}
	}
	return rows
}
