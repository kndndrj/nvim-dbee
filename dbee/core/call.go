package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neovim/go-client/msgpack"
)

type State int

const (
	CallStateUninitialized State = iota
	CallStateExecuting
	CallStateRetrieving
	CallStateArchived
	CallStateFailed
	CallStateCanceled
)

func StateFromString(s string) State {
	switch s {
	case "uninitialized":
		return CallStateUninitialized
	case "executing":
		return CallStateExecuting
	case "retrieving":
		return CallStateRetrieving
	case "archived":
		return CallStateArchived
	case "failed":
		return CallStateFailed
	case "canceled":
		return CallStateCanceled
	default:
		return CallStateUninitialized
	}
}

func (s State) String() string {
	switch s {
	case CallStateUninitialized:
		return "uninitialized"
	case CallStateExecuting:
		return "executing"
	case CallStateRetrieving:
		return "retrieving"
	case CallStateArchived:
		return "archived"
	case CallStateFailed:
		return "failed"
	case CallStateCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

type (
	StatID string

	Stat struct {
		ID        StatID
		Query     string
		State     State
		Took      time.Duration
		Timestamp time.Time

		result      *CacheResult
		archive     *archive
		cancelFunc  func()
		onEventFunc func(state State)
	}
)

// statPersistent is a form used to permanently store the call stat
type statPersistent struct {
	ID        string `msgpack:"id" json:"id"`
	Query     string `msgpack:"query" json:"query"`
	State     string `msgpack:"state" json:"state"`
	Took      int64  `msgpack:"took_us" json:"took_us"`
	Timestamp int64  `msgpack:"timestamp_us" json:"timestamp_us"`
}

func (s *Stat) toPersistent() *statPersistent {
	return &statPersistent{
		ID:        string(s.ID),
		Query:     s.Query,
		State:     s.State.String(),
		Took:      s.Took.Microseconds(),
		Timestamp: s.Timestamp.UnixMicro(),
	}
}

func (s *Stat) MarshalMsgPack(enc *msgpack.Encoder) error {
	return enc.Encode(s.toPersistent())
}

func (s *Stat) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toPersistent())
}

func (s *Stat) UnmarshalJSON(data []byte) error {
	var alias statPersistent

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	archive := newArchive(StatID(alias.ID))
	state := StateFromString(alias.ID)

	if !archive.isEmpty() {
		state = CallStateArchived
	}

	*s = Stat{
		ID:        StatID(alias.ID),
		Query:     alias.Query,
		State:     state,
		Took:      time.Duration(alias.Took) * time.Microsecond,
		Timestamp: time.UnixMicro(alias.Timestamp),

		result:  new(CacheResult),
		archive: newArchive(StatID(alias.ID)),
	}

	return nil
}

// Caller builds the cal
func newCallFromExecutor(executor func(context.Context) (IterResult, error), query string, onEvent func(state State)) *Stat {
	id := StatID(uuid.New().String())
	c := &Stat{
		ID:    id,
		Query: query,
		State: CallStateUninitialized,

		result:      new(CacheResult),
		archive:     newArchive(id),
		onEventFunc: onEvent,
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel

	go func() {
		c.Timestamp = time.Now()
		defer func() {
			c.Took = time.Since(c.Timestamp)
		}()

		// execute the function
		c.setState(CallStateExecuting)
		iter, err := executor(ctx)
		if err != nil {
			c.setState(CallStateFailed)
			return
		}
		c.setState(CallStateRetrieving)

		// set iterator to result
		err = c.result.setIter(iter)
		if err != nil {
			c.setState(CallStateFailed)
			return
		}

		// archive the result
		err = c.archive.setResult(c.result)
		if err != nil {
			c.setState(CallStateFailed)
			return
		}

		c.setState(CallStateArchived)
	}()

	return c
}

func (c *Stat) setState(state State) {
	if c.State == CallStateFailed || c.State == CallStateCanceled {
		return
	}
	c.State = state

	// trigger event callback
	if c.onEventFunc != nil {
		go c.onEventFunc(state)
	}
}

func (c *Stat) Cancel() {
	if c.State != CallStateExecuting {
		return
	}
	c.setState(CallStateCanceled)
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}

func (c *Stat) GetResult() (*CacheResult, error) {
	if c.result.IsEmpty() {
		iter, err := c.archive.getResult()
		if err != nil {
			return nil, fmt.Errorf("c.archive.getResult: %w", err)
		}
		err = c.result.setIter(iter)
		if err != nil {
			return nil, fmt.Errorf("c.result.setIter: %w", err)
		}
	}

	return c.result, nil
}
