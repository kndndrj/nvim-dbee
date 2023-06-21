package clients

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/redis/go-redis/v9"
)

// Register client
func init() {
	c := func(url string) (conn.Client, error) {
		return NewRedis(url)
	}
	_ = Store.Register("redis", c)

	// register known types with gob
	gob.Register(&redisResponse{})
	gob.Register([]any{})
	gob.Register(map[any]any{})
}

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

func (c *RedisClient) Query(query string) (models.IterResult, error) {
	cmd, err := parseRedisCmd(query)
	if err != nil {
		return nil, err
	}

	response, err := c.redis.Do(context.Background(), cmd...).Result()
	if err != nil {
		return nil, err
	}

	// iterator functions
	nextOnce := func(value any) func() (models.Row, error) {
		once := false
		return func() (models.Row, error) {
			if once {
				return nil, nil
			}
			once = true
			return models.Row{newRedisResponse(value)}, nil
		}
	}
	nextChannel := func(ch chan any) func() (models.Row, error) {
		return func() (models.Row, error) {
			val, ok := <-ch
			if !ok {
				return nil, nil
			}
			return models.Row{newRedisResponse(val)}, nil
		}
	}

	var nextFunc func() (models.Row, error)

	// parse response
	switch resp := response.(type) {
	case string, int64, map[any]any:
		nextFunc = nextOnce(resp)
	case []any:
		ch := make(chan any)
		go func() {
			defer close(ch)
			for _, item := range resp {
				ch <- item
			}
		}()
		nextFunc = nextChannel(ch)
	case nil:
		return nil, errors.New("no reponse from redis")
	default:
		return nil, fmt.Errorf("unknown type reponse from redis: %T", resp)
	}

	// build result
	result := common.NewResultBuilder().
		WithNextFunc(nextFunc).
		WithHeader(models.Header{"Reply"}).
		WithMeta(models.Meta{
			Query:      query,
			Timestamp:  time.Now(),
			SchemaType: models.SchemaLess,
		}).
		Build()

	return result, err
}

func (c *RedisClient) Layout() ([]models.Layout, error) {
	return []models.Layout{
		{
			Name:     "DB",
			Schema:   "",
			Database: "",
			Type:     models.LayoutTable,
		},
	}, nil
}

func (c *RedisClient) Close() {
	c.redis.Close()
}

// printSlice pretty prints nested slice using recursion
func printSlice(slice []any, level int) string {
	// indent prefix
	var prefix string
	for i := 0; i < level; i++ {
		prefix += "  "
	}

	var ret []string
	for _, v := range slice {
		if nested, ok := v.([]any); ok {
			ret = append(ret, printSlice(nested, level+1))
		} else {
			ret = append(ret, fmt.Sprintf("%s%v", prefix, v))
		}
	}
	return strings.Join(ret, "\n")
}

// printMap pretty prints map records
func printMap(m map[any]any) string {
	var ret []string
	for k, v := range m {
		ret = append(ret, fmt.Sprintf("%v: %v", k, v))
	}

	return strings.Join(ret, "\n")
}

// redisResponse serves as a wrapper around the mongo response
// to stringify the return values
type redisResponse struct {
	Value any
}

func newRedisResponse(val any) *redisResponse {
	return &redisResponse{
		Value: val,
	}
}

func (rr *redisResponse) String() string {
	switch value := rr.Value.(type) {
	case []any:
		return printSlice(value, 0)
	case map[any]any:
		return printMap(value)
	}
	return fmt.Sprint(rr.Value)
}

func (rr *redisResponse) MarshalJSON() ([]byte, error) {
	value := rr.Value

	m, ok := value.(map[any]any)
	if ok {
		ret := make(map[string]any)
		for k, v := range m {
			ret[fmt.Sprint(k)] = v
		}
		return json.Marshal(ret)
	}
	return json.Marshal(rr.Value)
}

// ErrUnmatchedDoubleQuote and ErrUnmatchedSingleQuote are errors returned from ParseRedisCmd
var (
	ErrUnmatchedDoubleQuote = func(position int) error { return fmt.Errorf("syntax error: unmatched double quote at: %d", position) }
	ErrUnmatchedSingleQuote = func(position int) error { return fmt.Errorf("syntax error: unmatched single quote at: %d", position) }
)

// parseRedisCmd parses string command into args for redis.Do
func parseRedisCmd(unparsed string) ([]any, error) {
	// error helper
	quoteErr := func(quote rune, position int) error {
		if quote == '"' {
			return ErrUnmatchedDoubleQuote(position)
		} else {
			return ErrUnmatchedSingleQuote(position)
		}
	}

	// return array
	var fields []any
	// what char is the current quote
	var blank rune
	var currentQuote struct {
		char     rune
		position int
	}
	// is the current char escaped or not?
	var escaped bool

	sb := &strings.Builder{}
	for i, r := range unparsed {
		// handle unescaped quotes
		if !escaped && (r == '"' || r == '\'') {
			// next char
			next := byte(' ')
			if i < len(unparsed)-1 {
				next = unparsed[i+1]
			}

			if r == currentQuote.char {
				if next != ' ' {
					return nil, quoteErr(r, i+1)
				}
				// end quote
				currentQuote.char = blank
				continue
			} else if currentQuote.char == blank {
				// start quote
				currentQuote.char = r
				currentQuote.position = i + 1
				continue
			}
		}

		// handle escapes
		if r == '\\' {
			escaped = true
			continue
		}

		// handle word end
		if currentQuote.char == blank && r == ' ' {
			fields = append(fields, sb.String())
			sb.Reset()
			continue
		}

		escaped = false
		sb.WriteRune(r)
	}

	// check if quote is not closed
	if currentQuote.char != blank {
		return nil, quoteErr(currentQuote.char, currentQuote.position)
	}

	// write last word
	if sb.Len() > 0 {
		fields = append(fields, sb.String())
	}

	return fields, nil
}
