package call

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
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

func (s CallState) String() string {
	switch s {
	case CallStateUninitialized:
		return "uninitialized"
	case CallStateExecuting:
		return "executing"
	case CallStateCaching:
		return "caching"
	case CallStateCached:
		return "cached"
	case CallStateArchived:
		return "archived"
	case CallStateFailed:
		return "failed"
	case CallStateCanceled:
		return "canceled"
	default:
		return ""
	}
}

type CallDetails struct {
	ID    string
	Query string
	State CallState
	Took  time.Duration
}

func (cd *CallDetails) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID    string `json:"id"`
		Query string `json:"query"`
		State string `json:"state"`
		Took  int64  `json:"took_ms"`
	}{
		ID:    cd.ID,
		Query: cd.Query,
		State: cd.State.String(),
		Took:  cd.Took.Milliseconds(),
	})
}

type stats struct {
	Took time.Duration
}

// Call represents a single call to database
// it contains various metadata fields, state and a context cancelation function
type Call struct {
	id     string
	state  CallState
	cancel func()
	cache  *cache
	log    models.Logger
	stats  *stats
}

func newCall(archivePath string, id string, logger models.Logger) *Call {
	return &Call{
		id:    id,
		state: CallStateUninitialized,
		cache: NewCache(archivePath, logger),
		log:   logger,
		stats: &stats{},
	}
}

func MakeCall(connID string, logger models.Logger, exec func(context.Context) (models.IterResult, error), callback func(*CallDetails)) *Call {
	id := uuid.New().String()
	c := &Call{
		id:    id,
		state: CallStateUninitialized,
		cache: NewCache(CallHistoryPath(connID, id), logger),
		log:   logger,
		stats: &stats{},
	}

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
		start := time.Now()

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

		c.stats.Took = time.Since(start)
		callback(c.GetDetails())
	}()

	return c
}

func (c *Call) GetDetails() *CallDetails {
	return &CallDetails{
		ID:    c.id,
		Query: "",
		State: c.state,
		Took:  c.stats.Took,
	}
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
