package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neovim/go-client/msgpack"
)

type CallState int

const (
	CallStateUnknown CallState = iota
	CallStateExecuting
	CallStateRetrieving
	CallStateArchived
	CallStateFailed
	CallStateCanceled
)

func CallStateFromString(s string) CallState {
	switch s {
	case "unknown":
		return CallStateUnknown
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
		return CallStateUnknown
	}
}

func (s CallState) String() string {
	switch s {
	case CallStateUnknown:
		return "unknown"
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
	CallID string

	Call struct {
		id        CallID
		query     string
		state     CallState
		timeTaken time.Duration
		timestamp time.Time

		result      *Result
		archive     *archive
		cancelFunc  func()
		onEventFunc func(state CallState)
	}
)

// callPersistent is a form used to permanently store the call stat
type callPersistent struct {
	ID        string `msgpack:"id" json:"id"`
	Query     string `msgpack:"query" json:"query"`
	State     string `msgpack:"state" json:"state"`
	TimeTaken int64  `msgpack:"time_taken_us" json:"time_taken_us"`
	Timestamp int64  `msgpack:"timestamp_us" json:"timestamp_us"`
}

func (s *Call) toPersistent() *callPersistent {
	return &callPersistent{
		ID:        string(s.id),
		Query:     s.query,
		State:     s.state.String(),
		TimeTaken: s.timeTaken.Microseconds(),
		Timestamp: s.timestamp.UnixMicro(),
	}
}

func (s *Call) MarshalMsgPack(enc *msgpack.Encoder) error {
	return enc.Encode(s.toPersistent())
}

func (s *Call) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toPersistent())
}

func (s *Call) UnmarshalJSON(data []byte) error {
	var alias callPersistent

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	archive := newArchive(CallID(alias.ID))
	state := CallStateFromString(alias.State)
	if state == CallStateArchived && archive.isEmpty() {
		state = CallStateUnknown
	}

	*s = Call{
		id:        CallID(alias.ID),
		query:     alias.Query,
		state:     state,
		timeTaken: time.Duration(alias.TimeTaken) * time.Microsecond,
		timestamp: time.UnixMicro(alias.Timestamp),

		result:  new(Result),
		archive: newArchive(CallID(alias.ID)),
	}

	return nil
}

// Caller builds the cal
func newCallFromExecutor(executor func(context.Context) (ResultStream, error), query string, onEvent func(CallState)) *Call {
	id := CallID(uuid.New().String())
	c := &Call{
		id:    id,
		query: query,
		state: CallStateUnknown,

		result:      new(Result),
		archive:     newArchive(id),
		onEventFunc: onEvent,
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel

	go func() {
		c.timestamp = time.Now()
		defer func() {
			c.timeTaken = time.Since(c.timestamp)
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

func (c *Call) GetID() CallID {
	return c.id
}

func (c *Call) GetQuery() string {
	return c.query
}

func (c *Call) GetState() CallState {
	return c.state
}

func (c *Call) GetTimeTaken() time.Duration {
	return c.timeTaken
}

func (c *Call) GetTimestamp() time.Time {
	return c.timestamp
}

func (c *Call) setState(state CallState) {
	if c.state == CallStateFailed || c.state == CallStateCanceled {
		return
	}
	c.state = state

	// trigger event callback
	if c.onEventFunc != nil {
		go c.onEventFunc(state)
	}
}

func (c *Call) Cancel() {
	if c.state != CallStateExecuting {
		return
	}
	c.setState(CallStateCanceled)
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}

func (c *Call) GetResult() (*Result, error) {
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
