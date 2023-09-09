package call

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// TODO:
type Output interface {
	Write(context.Context, models.Result) error
}

type CallState int

const (
	CallStateUninitialized CallState = iota
	CallStateExecuting
	CallStateCaching
	CallStateCached
	CallStateArchived
	CallStateFailed
	CallStateCanceled
)

// Call represents a single call to database
// it contains various metadata fields, state and a context cancelation function
type Call struct {
	id     string
	state  CallState
	cancel func()
	cache  *cache
	log    models.Logger
}

func MakeCall(id string, connID string, logger models.Logger) *Call {
	c := &Call{
		id:    id,
		state: CallStateUninitialized,
		cache: NewCache(id, logger),
		log:   logger,
	}
	return c
}

func (c *Call) Do(exec func(context.Context) (models.IterResult, error)) error {
	ctx, cancel := context.WithCancel(context.Background())

	if c.cancel != nil {
		oldCancel := c.cancel
		cancel = func() {
			oldCancel()
			cancel()
		}
	}
	c.cancel = cancel

	c.state = CallStateExecuting
	go func() {
		rows, err := exec(ctx)
		if err != nil {
			c.state = CallStateFailed
			if errors.Is(err, context.Canceled) {
				c.state = CallStateCanceled
			}
			c.log.Error(err.Error())
		}

		// save to cache
		c.state = CallStateCaching
		err = c.cache.Set(ctx, rows)
		if err != nil && !errors.Is(err, ErrCacheAlreadyFilled) {
			c.state = CallStateFailed
			if errors.Is(err, context.Canceled) {
				c.state = CallStateCanceled
			}
			c.log.Error(err.Error())
		}
	}()

	return nil
}

// GetResult pipes the selected range of rows to the outputs
// returns length of the result set
func (c *Call) GetResult(from int, to int, outputs ...Output) (int, error) {
	ctx := context.TODO()

	return c.cache.Get(ctx, from, to, outputs...)
}

func (c *Call) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (s *Call) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID    string `json:"id"`
		State int    `json:"state"`
	}{
		ID:    s.id,
		State: int(s.state),
	})
}
