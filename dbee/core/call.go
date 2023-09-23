package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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
		onEventFunc func(*Call)

		// any error that might occur during execution
		err error
	}
)

// callPersistent is used for marshaling and unmarshaling the call
type callPersistent struct {
	ID        string `json:"id"`
	Query     string `json:"query"`
	State     string `json:"state"`
	TimeTaken int64  `json:"time_taken_us"`
	Timestamp int64  `json:"timestamp_us"`
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
func newCallFromExecutor(executor func(context.Context) (ResultStream, error), query string, onEvent func(*Call)) *Call {
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
			c.err = err
			c.setState(CallStateFailed)
			return
		}

		// set iterator to result
		err = c.result.setIter(iter, func() { c.setState(CallStateRetrieving) })
		if err != nil {
			c.err = err
			c.setState(CallStateFailed)
			return
		}

		// archive the result
		err = c.archive.setResult(c.result)
		if err != nil {
			c.err = err
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

func (c *Call) Err() error {
	return c.err
}

func (c *Call) setState(state CallState) {
	if c.state == CallStateFailed || c.state == CallStateCanceled {
		return
	}
	c.state = state

	// trigger event callback
	if c.onEventFunc != nil {
		c.onEventFunc(c)
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
		err = c.result.setIter(iter, nil)
		if err != nil {
			return nil, fmt.Errorf("c.result.setIter: %w", err)
		}
	}

	return c.result, nil
}
