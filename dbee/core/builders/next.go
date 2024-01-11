package builders

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// NextSingle creates next and hasNext functions from a provided single value
func NextSingle(value any) (func() (core.Row, error), func() bool) {
	has := true

	// iterator functions
	next := func() (core.Row, error) {
		if !has {
			return nil, errors.New("no next row")
		}
		has = false
		return core.Row{value}, nil
	}

	hasNext := func() bool {
		return has
	}

	return next, hasNext
}

// NextSlice creates next and hasNext functions from provided values
// preprocessor is an optional function which parses a single value from slice before adding it to a row
func NextSlice[T any](values []T, preprocess func(T) any) (func() (core.Row, error), func() bool) {
	if preprocess == nil {
		preprocess = func(v T) any { return v }
	}

	index := 0

	hasNext := func() bool {
		return index < len(values)
	}

	// iterator functions
	next := func() (core.Row, error) {
		if !hasNext() {
			return nil, errors.New("no next row")
		}

		row := core.Row{preprocess(values[index])}
		index++
		return row, nil
	}

	return next, hasNext
}

// NextNil creates next and hasNext functions that don't return anything (no rows)
func NextNil() (func() (core.Row, error), func() bool) {
	hasNext := func() bool {
		return false
	}

	// iterator functions
	next := func() (core.Row, error) {
		return nil, errors.New("no next row")
	}

	return next, hasNext
}

// closeOnce closes the channel if it isn't already closed.
func closeOnce[T any](ch chan T) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// NextYield creates next and hasNext functions by calling yield in internal function.
// WARNING: the caller must call "hasNext" before each call to "next".
func NextYield(fn func(yield func(...any)) error) (func() (core.Row, error), func() bool) {
	resultsCh := make(chan []any, 10)
	errorsCh := make(chan error, 1)
	readyCh := make(chan struct{})
	doneCh := make(chan struct{})

	// spawn channel function
	go func() {
		defer func() {
			close(doneCh)
			closeOnce(readyCh)
			close(resultsCh)
			close(errorsCh)
		}()

		err := fn(func(v ...any) {
			resultsCh <- v
			closeOnce(readyCh)
		})
		if err != nil {
			errorsCh <- err
		}
	}()

	<-readyCh

	var nextVal atomic.Value
	var nextErr atomic.Value

	var hasNext func() bool
	hasNext = func() bool {
		select {
		case vals, ok := <-resultsCh:
			if !ok {
				return false
			}
			nextVal.Store(vals)
			return true
		case err := <-errorsCh:
			if err != nil {
				nextErr.Store(err)
				return false
			}
		case <-doneCh:
			if len(resultsCh) < 1 {
				return false
			}
		case <-time.After(5 * time.Second):
			nextErr.Store(errors.New("next row timeout"))
			return false
		}

		return hasNext()
	}

	next := func() (core.Row, error) {
		var val core.Row
		var err error

		nval := nextVal.Load()
		if nval != nil {
			val = nval.([]any)
		}
		nerr := nextErr.Load()
		if nerr != nil {
			err = nerr.(error)
		}
		return val, err
	}

	return next, hasNext
}
