package core

import (
	"context"
	"encoding/json"
	"errors"
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

		result     *Result
		archive    *archive
		cancelFunc func()

		// any error that might occur during execution
		err  error
		done chan struct{}
	}
)

// callPersistent is used for marshaling and unmarshaling the call
type callPersistent struct {
	ID        string `json:"id"`
	Query     string `json:"query"`
	State     string `json:"state"`
	TimeTaken int64  `json:"time_taken_us"`
	Timestamp int64  `json:"timestamp_us"`
	Error     string `json:"error,omitempty"`
}

func (c *Call) toPersistent() *callPersistent {
	errMsg := ""
	if c.err != nil {
		errMsg = c.err.Error()
	}

	return &callPersistent{
		ID:        string(c.id),
		Query:     c.query,
		State:     c.state.String(),
		TimeTaken: c.timeTaken.Microseconds(),
		Timestamp: c.timestamp.UnixMicro(),
		Error:     errMsg,
	}
}

func (s *Call) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.toPersistent())
}

func (c *Call) UnmarshalJSON(data []byte) error {
	var alias callPersistent

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	done := make(chan struct{})
	close(done)

	archive := newArchive(CallID(alias.ID))
	state := CallStateFromString(alias.State)
	if state == CallStateArchived && archive.isEmpty() {
		state = CallStateUnknown
	}

	var callErr error
	if alias.Error != "" {
		callErr = errors.New(alias.Error)
	}

	*c = Call{
		id:        CallID(alias.ID),
		query:     alias.Query,
		state:     state,
		timeTaken: time.Duration(alias.TimeTaken) * time.Microsecond,
		timestamp: time.UnixMicro(alias.Timestamp),
		err:       callErr,

		result:  new(Result),
		archive: newArchive(CallID(alias.ID)),

		done: done,
	}

	return nil
}

func newCallFromExecutor(executor func(context.Context) (ResultStream, error), query string, onEvent func(CallState, *Call)) *Call {
	id := CallID(uuid.New().String())
	c := &Call{
		id:    id,
		query: query,
		state: CallStateUnknown,

		result:  new(Result),
		archive: newArchive(id),

		done: make(chan struct{}),
	}

	eventsCh := make(chan CallState, 10)

	ctx, cancel := context.WithCancel(context.Background())
	c.timestamp = time.Now()
	c.cancelFunc = func() {
		cancel()
		c.timeTaken = time.Since(c.timestamp)
		eventsCh <- CallStateCanceled
	}

	// event function handler
	go func() {
		for state := range eventsCh {
			if c.state == CallStateExecutingFailed ||
				c.state == CallStateRetrievingFailed ||
				c.state == CallStateCanceled {
				return
			}
			c.state = state

			// trigger event callback
			if onEvent != nil {
				onEvent(state, c)
			}
		}
	}()

	go func() {
		defer close(eventsCh)

		// execute the function
		eventsCh <- CallStateExecuting
		iter, err := executor(ctx)
		if err != nil {
			c.timeTaken = time.Since(c.timestamp)
			c.err = err
			eventsCh <- CallStateExecutingFailed
			close(c.done)
			return
		}

		// set iterator to result
		err = c.result.SetIter(iter, func() { eventsCh <- CallStateRetrieving })
		if err != nil {
			c.timeTaken = time.Since(c.timestamp)
			c.err = err
			eventsCh <- CallStateRetrievingFailed
			close(c.done)
			return
		}

		// archive the result
		err = c.archive.setResult(c.result)
		if err != nil {
			c.timeTaken = time.Since(c.timestamp)
			c.err = err
			eventsCh <- CallStateArchiveFailed
			close(c.done)
			return
		}

		c.timeTaken = time.Since(c.timestamp)
		eventsCh <- CallStateArchived
		close(c.done)
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

// Done returns a non-buffered channel that is closed when
// call finishes.
func (c *Call) Done() chan struct{} {
	return c.done
}

func (c *Call) Cancel() {
	if c.state > CallStateExecuting {
		return
	}
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
		err = c.result.SetIter(iter, nil)
		if err != nil {
			return nil, fmt.Errorf("c.result.setIter: %w", err)
		}
	}

	return c.result, nil
}
