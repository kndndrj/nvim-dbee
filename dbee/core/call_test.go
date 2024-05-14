package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/mock"
)

func TestCall_Success(t *testing.T) {
	r := require.New(t)

	rows := mock.NewRows(0, 10)

	connection, err := core.NewConnection(&core.ConnectionParams{}, mock.NewAdapter(rows,
		mock.AdapterWithResultStreamOpts(mock.ResultStreamWithNextSleep(300*time.Millisecond)),
	))
	r.NoError(err)

	expectedEvents := []core.CallState{
		core.CallStateExecuting,
		core.CallStateRetrieving,
		core.CallStateArchived,
	}

	eventIndex := 0
	call := connection.Execute("_", func(state core.CallState, c *core.Call) {
		// make sure events were in order
		r.Equal(expectedEvents[eventIndex], state)
		eventIndex++

		if state == core.CallStateRetrieving {
			result, err := c.GetResult()
			r.NoError(err)

			actualRows, err := result.Rows(0, len(rows))
			r.NoError(err)

			r.Equal(rows, actualRows)
		}
	})

	// wait for call to finish
	select {
	case <-call.Done():
		// wait a bit for event index to stabilize
		time.Sleep(100 * time.Millisecond)
	case <-time.After(5 * time.Second):
		t.Error("call did not finish in expected time")
	}

	// make sure all events passed
	r.Equal(len(expectedEvents), eventIndex)
}

func TestCall_Cancel(t *testing.T) {
	r := require.New(t)

	rows := mock.NewRows(0, 10)

	adapter := mock.NewAdapter(rows,
		mock.AdapterWithQuerySideEffect("wait", func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second):
			}
			return nil
		}),
		mock.AdapterWithResultStreamOpts(mock.ResultStreamWithNextSleep(300*time.Millisecond)),
	)

	connection, err := core.NewConnection(&core.ConnectionParams{}, adapter)
	r.NoError(err)

	expectedEvents := []core.CallState{
		core.CallStateExecuting,
		core.CallStateCanceled,
	}

	eventIndex := 0
	call := connection.Execute("wait", func(state core.CallState, c *core.Call) {
		// wait for first event and cancel request
		c.Cancel()
		// make sure events were in order
		r.Equal(expectedEvents[eventIndex], state)
		eventIndex++
	})

	// wait for call to finish
	select {
	case <-call.Done():
		// wait a bit for event index to stabilize
		time.Sleep(100 * time.Millisecond)
	case <-time.After(5 * time.Second):
		t.Error("call did not finish in expected time")
	}

	// make sure all events passed
	r.Equal(len(expectedEvents), eventIndex)
}

func TestCall_FailedQuery(t *testing.T) {
	r := require.New(t)

	rows := mock.NewRows(0, 10)

	adapter := mock.NewAdapter(rows,
		mock.AdapterWithQuerySideEffect("fail", func(ctx context.Context) error {
			return errors.New("query failed")
		}),
		mock.AdapterWithResultStreamOpts(mock.ResultStreamWithNextSleep(300*time.Millisecond)),
	)

	connection, err := core.NewConnection(&core.ConnectionParams{}, adapter)
	r.NoError(err)

	expectedEvents := []core.CallState{
		core.CallStateExecuting,
		core.CallStateExecutingFailed,
	}

	eventIndex := 0
	call := connection.Execute("fail", func(state core.CallState, c *core.Call) {
		// make sure events were in order
		r.Equal(expectedEvents[eventIndex], state)
		eventIndex++

		if state == core.CallStateExecutingFailed {
			r.NotNil(c.Err())
		}
	})

	// wait for call to finish
	select {
	case <-call.Done():
		// wait a bit for event index to stabilize
		time.Sleep(100 * time.Millisecond)
	case <-time.After(5 * time.Second):
		t.Error("call did not finish in expected time")
	}

	// make sure all events passed
	r.Equal(len(expectedEvents), eventIndex)
}

func TestCall_Archive(t *testing.T) {
	r := require.New(t)

	rows := mock.NewRows(0, 10)

	connection, err := core.NewConnection(&core.ConnectionParams{}, mock.NewAdapter(rows,
		mock.AdapterWithResultStreamOpts(mock.ResultStreamWithNextSleep(300*time.Millisecond)),
	))
	r.NoError(err)

	call := connection.Execute("_", nil)

	// wait for call to finish
	select {
	case <-call.Done():
		// wait a bit for event index to stabilize
		time.Sleep(100 * time.Millisecond)
	case <-time.After(5 * time.Second):
		t.Error("call did not finish in expected time")
	}

	// check result
	result, err := call.GetResult()
	r.NoError(err)
	actualRows, err := result.Rows(0, len(rows))
	r.NoError(err)
	r.Equal(rows, actualRows)

	// marshal to json
	b, err := json.Marshal(call)
	r.NoError(err)

	// marshal back
	restoredCall := new(core.Call)
	err = json.Unmarshal(b, restoredCall)
	r.NoError(err)

	// check result again
	result, err = restoredCall.GetResult()
	r.NoError(err)
	actualRows, err = result.Rows(0, len(rows))
	r.NoError(err)
	r.Equal(rows, actualRows)
}
