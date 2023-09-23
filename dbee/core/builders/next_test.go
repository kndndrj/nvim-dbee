package builders_test

import (
	"errors"
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
)

func testNextYield(t *testing.T, sleep bool) {
	vals := []string{"here", "are", "some", "random", "values"}

	next, hasNext := builders.NextYield(func(yield func(any)) error {
		for i, val := range vals {
			if sleep && (i == 2 || i == 4) {
				time.Sleep(500 * time.Millisecond)
			}
			yield(val)
		}

		return nil
	})

	i := 0
	for hasNext() {
		val, err := next()
		assert.NilError(t, err)

		if len(val) < 1 {
			t.Fatal("row without value")
		}

		assert.Equal(t, val[0], vals[i])
		i++
	}

	assert.Equal(t, i, len(vals))
}

func TestNextYield_Success(t *testing.T) {
	// test with random sleeping
	testNextYield(t, true)

	for i := 0; i < 1000; i++ {
		testNextYield(t, false)
	}
}

func TestNextYield_Error(t *testing.T) {
	expectedError := errors.New("expected error")

	next, hasNext := builders.NextYield(func(yield func(any)) error {
		return expectedError
	})

	for hasNext() {
		_, err := next()
		assert.Error(t, err, expectedError.Error())
	}
}
