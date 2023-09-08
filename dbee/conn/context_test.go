package conn

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestCallContextUpdateState(t *testing.T) {
	updatedState := CallStateArchived

	call := &Call{
		ID:    "id",
		Query: "query",
		State: CallStateExecuting,
	}

	ctx, _ := newCallContext(call)

	err := contextUpdateCallState(ctx, updatedState)
	assert.NilError(t, err)

	// check if original state changed
	assert.Equal(t, call.State, updatedState)
}

func TestCallContextCancel(t *testing.T) {
	call := &Call{
		ID:    "id",
		Query: "query",
		State: CallStateExecuting,
	}

	ctx, cancel := newCallContext(call)
	call.Cancel = cancel

	canceled := false
	go func() {
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				canceled = true
				return
			default:
			}

			time.Sleep(500 * time.Millisecond)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	call.Cancel()

	time.Sleep(1000 * time.Millisecond)

	assert.Equal(t, canceled, true)
}
