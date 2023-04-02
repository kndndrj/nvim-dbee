package clients

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

	meta := conn.Meta{
		Query:     query,
		Timestamp: time.Now(),
	}

	redisRows, err := newRedisRows(resp, meta)

	return redisRows, err
}

func (c *RedisClient) Schema() (conn.Schema, error) {
	return conn.Schema{
		"DB": []string{"DB"},
	}, nil
}

func (c *RedisClient) Close() {
	c.redis.Close()
}

type RedisRows struct {
	iter func() conn.Row
	meta conn.Meta
}

func newRedisRows(reply any, meta conn.Meta) (*RedisRows, error) {
	iter, err := getIter(reply)
	if err != nil {
		return nil, err
	}

	return &RedisRows{
		iter: iter,
		meta: meta,
	}, nil
}

func (r *RedisRows) Meta() (conn.Meta, error) {
	return r.meta, nil
}

func (r *RedisRows) Header() (conn.Header, error) {
	return conn.Header{"Reply"}, nil
}

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

func getIter(redisReply any) (func() conn.Row, error) {

	var rows []conn.Row
	switch rpl := redisReply.(type) {
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

	max := len(rows) - 1
	i := 0
	return func() conn.Row {
		if i > max {
			return nil
		}
		val := rows[i]
		i++
		return val
	}, nil
}

func (r *RedisRows) Next() (conn.Row, error) {
	return r.iter(), nil
}

func (r *RedisRows) Close() {
}
