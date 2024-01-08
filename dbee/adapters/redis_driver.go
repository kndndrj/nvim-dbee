package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

var _ core.Driver = (*redisDriver)(nil)

type redisDriver struct {
	redis *redis.Client
}

func redisResponseToNext(response any) (func() (core.Row, error), func() bool) {
	// parse response
	switch resp := response.(type) {
	case string, int64, map[any]any:
		return builders.NextSingle(newRedisResponse(resp))
	case []any:
		return builders.NextSlice(resp, newRedisResponse)
	default:
		return builders.NextNil()
	}
}

func (c *redisDriver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	cmd, err := parseRedisCmd(query)
	if err != nil {
		return nil, err
	}

	response, err := c.redis.Do(ctx, cmd...).Result()
	if err != nil {
		return nil, err
	}

	next, hasNext := redisResponseToNext(response)

	// build result
	result := builders.NewResultStreamBuilder().
		WithNextFunc(next, hasNext).
		WithHeader(core.Header{"Reply"}).
		WithMeta(&core.Meta{
			SchemaType: core.SchemaLess,
		}).
		Build()

	return result, err
}

func (c *redisDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	return []*core.Column{
		{
			Name: "",
			Type: "redis",
		},
	}, nil
}

func (c *redisDriver) Structure() ([]*core.Structure, error) {
	return []*core.Structure{
		{
			Name:   "Storage",
			Schema: "",
			Type:   core.StructureTypeTable,
		},
	}, nil
}

func (c *redisDriver) Close() {
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

// redisResponse serves as a wrapper around the redis response
// to stringify the return values
type redisResponse struct {
	Value any
}

// a preprocessor for redis response
func newRedisResponse(val any) any {
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
