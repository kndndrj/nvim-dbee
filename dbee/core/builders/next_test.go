package builders_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	"github.com/stretchr/testify/require"
)

func testNextYield(t *testing.T, sleep bool) {
	r := require.New(t)

	rows := [][]any{{"first", "row"}, {"second"}, {"third"}, {"fourth"}, {"fifth"}, {"and", "last", "row"}}

	next, hasNext := builders.NextYield(func(yield func(...any)) error {
		for i, row := range rows {
			if sleep && (i == 2 || i == 4) {
				time.Sleep(500 * time.Millisecond)
			}
			yield(row...)
		}

		return nil
	})

	i := 0
	for hasNext() {
		row, err := next()

		r.NoError(err)

		r.NotEqual(0, len(row))

		r.Equal(row, core.Row(rows[i]))

		i++
	}

	r.Equal(i, len(rows))
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

	next, hasNext := builders.NextYield(func(yield func(...any)) error {
		return expectedError
	})

	for hasNext() {
		_, err := next()
		require.Error(t, err, expectedError.Error())
	}
}

func TestNextYield_NoRows(t *testing.T) {
	_, hasNext := builders.NextYield(func(yield func(...any)) error {
		time.Sleep(1 * time.Second)
		return nil
	})

	require.Equal(t, false, hasNext())
}

func TestNextYield_SingleRow(t *testing.T) {
	r := require.New(t)
	next, hasNext := builders.NextYield(func(yield func(...any)) error {
		yield(1)
		time.Sleep(1 * time.Second)
		return nil
	})

	r.True(hasNext())

	row, err := next()
	r.NoError(err)
	r.Equal(1, len(row))
	r.Equal(1, row[0])

	r.Equal(false, hasNext())
}
