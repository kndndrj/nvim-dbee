package call

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/neovim/go-client/msgpack"
)

type CallState int

const (
	CallStateUninitialized CallState = iota
	CallStateExecuting
	CallStateCached
	CallStateArchived
	CallStateFailed
)

func (s CallState) String() string {
	switch s {
	case CallStateUninitialized:
		return "uninitialized"
	case CallStateExecuting:
		return "executing"
	case CallStateCached:
		return "cached"
	case CallStateArchived:
		return "archived"
	case CallStateFailed:
		return "failed"
	default:
		return ""
	}
}

type CallDetails struct {
	ID          string
	Query       string
	State       CallState
	Took        time.Duration
	Timestamp   time.Time
	ArchivePath string
}

func (cd *CallDetails) MarshalMsgPack(enc *msgpack.Encoder) error {
	return enc.Encode(&struct {
		ID          string `msgpack:"id"`
		Query       string `msgpack:"query"`
		State       string `msgpack:"state"`
		Took        int64  `msgpack:"took_ms"`
		Timestamp   int64  `msgpack:"timestamp"`
		ArchivePath string `msgpack:"archive_path"`
	}{
		ID:          cd.ID,
		Query:       cd.Query,
		State:       cd.State.String(),
		Took:        cd.Took.Microseconds(),
		Timestamp:   cd.Timestamp.UnixMicro(),
		ArchivePath: cd.ArchivePath,
	})
}

// Caller builds the call
type Caller struct {
	id          string
	log         models.Logger
	query       string
	exec        func(context.Context) (models.IterResult, error)
	callback    func(*CallDetails)
	archivePath string
}

func NewCaller(logger models.Logger) *Caller {
	id := uuid.New().String()
	return &Caller{
		id:          id,
		log:         logger,
		callback:    func(*CallDetails) {},
		archivePath: callHistoryPath(id),
	}
}

func (b *Caller) WithID(id string) *Caller {
	b.id = id
	return b
}

func (b *Caller) WithArchivePath(path string) *Caller {
	b.archivePath = path
	return b
}

func (b *Caller) WithQuery(query string) *Caller {
	b.query = query
	return b
}

func (b *Caller) WithExecutor(executor func(context.Context) (models.IterResult, error)) *Caller {
	b.exec = executor
	return b
}

func (b *Caller) WithCallback(cb func(*CallDetails)) *Caller {
	b.callback = cb
	return b
}

func (b *Caller) FromDetails(details *CallDetails) *Call {
	details.State = CallStateUninitialized
	return &Call{
		cache:   NewCache(details.ArchivePath, b.log),
		log:     b.log,
		details: details,
	}
}

func (b *Caller) Do() *Call {
	c := &Call{
		cache: NewCache(b.archivePath, b.log),
		log:   b.log,
		details: &CallDetails{
			ID:          b.id,
			Query:       b.query,
			State:       CallStateUninitialized,
			ArchivePath: b.archivePath,
		},
	}

	if b.exec == nil {
		return c
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	c.details.State = CallStateExecuting
	go func() {
		c.details.Timestamp = time.Now()
		defer func() {
			c.details.Took = time.Since(c.details.Timestamp)
			b.callback(c.GetDetails())
		}()

		rows, err := b.exec(ctx)
		if err != nil {
			c.details.State = CallStateFailed
			c.log.Error(err.Error())
			return
		}

		// save to cache
		err = c.cache.Set(ctx, rows)
		if err != nil && !errors.Is(err, ErrCacheAlreadyFilled) {
			c.details.State = CallStateFailed
			c.log.Error(err.Error())
			return
		}
	}()

	return c
}

// Call represents a single call to database
// it contains various metadata fields, state and a context cancelation function
type Call struct {
	cancel  func()
	cache   *cache
	log     models.Logger
	details *CallDetails
}

func (c *Call) GetDetails() *CallDetails {
	// update state from cache
	if c.cache.HasResultInMemory() {
		c.details.State = CallStateCached
	} else if c.cache.HasResultInArchive() {
		c.details.State = CallStateArchived
	}

	return c.details
}

// GetResult pipes the selected range of rows to the outputs
// returns length of the result set
func (c *Call) GetResult(from int, to int) (models.IterResult, error) {
	ctx, cancel := context.WithCancel(context.Background())
	oldCancel := c.cancel
	c.cancel = func() {
		if oldCancel != nil {
			oldCancel()
		}
		cancel()
		c.cancel = nil
	}

	return c.cache.Get(ctx, from, to)
}

func (c *Call) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}
