package conn

import (
	"context"
	"errors"
)

type contextKey string

const callContextKey contextKey = "__call_context_key__"

func newCallContext(call *Call) (context.Context, context.CancelFunc) {
	valueContext := context.WithValue(context.Background(), callContextKey, call)
	return context.WithCancel(valueContext)
}

func contextUpdateCallState(ctx context.Context, state CallState) error {
	val := ctx.Value(callContextKey)
	if val == nil {
		return errors.New("call does not exist in this context")
	}

	call, ok := val.(*Call)
	if !ok {
		return errors.New("could not extract call from the context")
	}

	// update the state
	call.State = state
	return nil
}

func contextGetCall(ctx context.Context) (*Call, error) {
	val := ctx.Value(callContextKey)
	if val == nil {
		return nil, errors.New("call does not exist in this context")
	}

	call, ok := val.(*Call)
	if !ok {
		return nil, errors.New("could not extract call from the context")
	}

	// update the state
	return call, nil
}
