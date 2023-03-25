package clients

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

func (c *RedisClient) Execute(query string) (Rows, error) {

	q := strings.Split(query, " ")

	args := make([]any, len(q))
	for i, v := range q {
		args[i] = v
	}

	resp, err := c.redis.Do(context.Background(), args...).Result()
	if err != nil {
		return nil, err
	}

	redisRows, err := NewRedisRows(resp)

	return redisRows, err
}

func (c *RedisClient) Schema() (Schema, error) {
	return Schema{
		"DB": []string{"DB"},
	}, nil
}

func (c *RedisClient) Close() {
	c.redis.Close()
}

type RedisRows struct {
	iter func() Row
}

func NewRedisRows(reply any) (*RedisRows, error) {
	iter, err := getIter(reply)
	if err != nil {
		return nil, err
	}

	return &RedisRows{
		iter: iter,
	}, nil
}

func (r *RedisRows) Header() (Header, error) {
	return Header{"Reply"}, nil
}

func sliceToRows(slice []any, level int) []Row {
	var rows []Row

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

func getIter(redisReply any) (func() Row, error) {

	var rows []Row
	switch rpl := redisReply.(type) {
	case int64:
		rows = []Row{{rpl}}
	case string:
		rows = []Row{{rpl}}
	case []any:
		rows = sliceToRows(rpl, -1)
	case map[any]any:
		for k, v := range rpl {
			rows = append(rows, Row{k, v})
		}
	case nil:
		return nil, errors.New("no reponse from redis")
	default:
		return nil, fmt.Errorf("unknown type reponse from redis: %T!\n", rpl)
	}

	max := len(rows) - 1
	i := 0
	return func() Row {
		if i > max {
			return nil
		}
		val := rows[i]
		i++
		return val
	}, nil
}

func (r *RedisRows) Next() (Row, error) {
	return r.iter(), nil
}
