package builders

import (
	"errors"
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

// NextYield creates next and hasNext functions from provided values
// preprocessor is an optional function which parses a single value from slice before adding it to a row
func NextYield(fn func(yield func(any)) error) (func() (core.Row, error), func() bool) {
	ch := make(chan any, 10)
	errCh := make(chan error, 1)

	// spawn channel function
	done := false
	go func() {
		err := fn(func(v any) {
			ch <- v
		})
		if err != nil {
			errCh <- err
		}
		close(ch)
		close(errCh)
		done = true
	}()

	hasNext := func() bool {
		if done && len(ch) < 1 {
			return false
		}
		return true
	}

	var next func() (core.Row, error)
	next = func() (core.Row, error) {
		if !hasNext() {
			return nil, errors.New("no next row")
		}

		// read value channel
		select {
		case val := <-ch:
			return core.Row{val}, nil
		case <-time.After(5 * time.Second):
			return nil, errors.New("next row timeout")
		case err := <-errCh:
			if err != nil {
				return nil, err
			}
			return next()
		}
	}

	return next, hasNext
}
