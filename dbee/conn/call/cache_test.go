package call

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
	"gotest.tools/assert"
)

type mockLogger struct{}

func (ml *mockLogger) Debug(msg string) {
	log.Default().Print(msg)
}

func (ml *mockLogger) Debugf(format string, args ...any) {
	log.Default().Printf(format, args...)
}

func (ml *mockLogger) Info(msg string) {
	log.Default().Print(msg)
}

func (ml *mockLogger) Infof(format string, args ...any) {
	log.Default().Printf(format, args...)
}

func (ml *mockLogger) Warn(msg string) {
	log.Default().Print(msg)
}

func (ml *mockLogger) Warnf(format string, args ...any) {
	log.Default().Printf(format, args...)
}

func (ml *mockLogger) Error(msg string) {
	log.Default().Print(msg)
}

func (ml *mockLogger) Errorf(format string, args ...any) {
	log.Default().Printf(format, args...)
}

type mockedIterResult struct {
	max     int
	current int
	sleep   time.Duration
}

func newMockedIterResult(maxRows int, sleep time.Duration) *mockedIterResult {
	return &mockedIterResult{
		max:   maxRows,
		sleep: sleep,
	}
}

func (mir *mockedIterResult) Meta() (models.Meta, error) {
	return models.Meta{}, nil
}

func (mir *mockedIterResult) Header() (models.Header, error) {
	return models.Header{"header1", "header2"}, nil
}

func (mir *mockedIterResult) Next() (models.Row, error) {
	if mir.current < mir.max {

		// sleep between iterations
		time.Sleep(mir.sleep)

		num := mir.current
		mir.current += 1
		return models.Row{num, strconv.Itoa(num)}, nil
	}

	return nil, nil
}

func (mir *mockedIterResult) Close() {
}

func (mir *mockedIterResult) Range(from int, to int) []models.Row {
	var rows []models.Row

	for i := from; i < to; i++ {
		rows = append(rows, models.Row{i, strconv.Itoa(i)})
	}
	return rows
}

func TestCache(t *testing.T) {
	// prepare cache and mocks
	cache := NewCache("", &mockLogger{})

	numOfRows := 10
	rows := newMockedIterResult(numOfRows, 0)

	err := cache.Set(context.Background(), rows)
	assert.NilError(t, err)

	type testCase struct {
		name           string
		from           int
		to             int
		before         func()
		expectedResult []models.Row
		expectedError  error
	}

	testCases := []testCase{
		{
			name:           "get all",
			from:           0,
			to:             -1,
			expectedResult: rows.Range(0, numOfRows),
			expectedError:  nil,
		},
		{
			name:           "get basic range",
			from:           0,
			to:             3,
			expectedResult: rows.Range(0, 3),
			expectedError:  nil,
		},
		{
			name:           "get last 2",
			from:           -3,
			to:             -1,
			expectedResult: rows.Range(numOfRows-2, numOfRows),
			expectedError:  nil,
		},
		{
			name:           "get only one",
			from:           0,
			to:             1,
			expectedResult: rows.Range(0, 1),
			expectedError:  nil,
		},

		{
			name:           "invalid range",
			from:           5,
			to:             1,
			expectedResult: nil,
			expectedError:  ErrInvalidRange(5, 1),
		},
		{
			name:           "invalid range (even if 10 can be higher than -1, its undefined and should fail)",
			from:           -5,
			to:             10,
			expectedResult: nil,
			expectedError:  ErrInvalidRange(-5, 10),
		},

		{
			name:           "wait for available index",
			from:           0,
			to:             3,
			expectedResult: rows.Range(0, 3),
			expectedError:  nil,
			before: func() {
				cache.Wipe()
				// reset result with sleep between iterations
				err = cache.Set(context.Background(), newMockedIterResult(numOfRows, 500*time.Millisecond))
				assert.NilError(t, err)
			},
		},
		{
			name:           "wait for all to be drained",
			from:           0,
			to:             -1,
			expectedResult: rows.Range(0, numOfRows),
			expectedError:  nil,
			before: func() {
				cache.Wipe()
				// reset result with sleep between iterations
				err = cache.Set(context.Background(), newMockedIterResult(numOfRows, 500*time.Millisecond))
				assert.NilError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.before != nil {
				tc.before()
			}

			result, err := cache.Get(context.Background(), tc.from, tc.to)
			if err != nil && tc.expectedError != nil {
				assert.Equal(t, err.Error(), tc.expectedError.Error())
				return
			}

			// drain the iterator and compare results
			var resultRows []models.Row

			for {
				row, err := result.Next()
				assert.NilError(t, err)
				if row == nil {
					break
				}
				resultRows = append(resultRows, row)
			}
			fmt.Println(resultRows)
			fmt.Println(tc.expectedResult)

			assert.DeepEqual(t, tc.expectedResult, resultRows)
		})
	}
}
