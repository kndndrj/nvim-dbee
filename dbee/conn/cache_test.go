package conn_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
	"gotest.tools/assert"
)

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

type mockOutput struct {
	t        *testing.T
	expected []models.Row
}

func newMockOutput(t *testing.T) *mockOutput {
	return &mockOutput{
		t: t,
	}
}

// expect this result on next write
func (mo *mockOutput) expect(result []models.Row) {
	mo.expected = result
}

func (mo *mockOutput) Write(_ context.Context, result models.Result) error {
	if mo.expected != nil {
		assert.DeepEqual(mo.t, mo.expected, result.Rows)
	}

	return nil
}

func TestCache(t *testing.T) {
	// prepare cache and mocks
	cache := conn.NewCache()

	numOfRows := 10
	rows := newMockedIterResult(numOfRows, 0)

	err := cache.Set(context.Background(), rows)
	assert.NilError(t, err)

	type testCase struct {
		from           int
		to             int
		before         func()
		expectedResult []models.Row
		expectedError  error
	}

	testCases := []testCase{
		// get all
		{
			from:           0,
			to:             -1,
			expectedResult: rows.Range(0, numOfRows),
			expectedError:  nil,
		},
		// get basic range
		{
			from:           0,
			to:             3,
			expectedResult: rows.Range(0, 3),
			expectedError:  nil,
		},
		// get last 2
		{
			from:           -3,
			to:             -1,
			expectedResult: rows.Range(numOfRows-2, numOfRows),
			expectedError:  nil,
		},
		// get only one
		{
			from:           0,
			to:             1,
			expectedResult: rows.Range(0, 1),
			expectedError:  nil,
		},

		// invalid range
		{
			from:           5,
			to:             1,
			expectedResult: nil,
			expectedError:  conn.ErrInvalidRange(5, 1),
		},
		// invalid range (even if 10 can be higher than -1, its undefined and should fail)
		{
			from:           -5,
			to:             10,
			expectedResult: nil,
			expectedError:  conn.ErrInvalidRange(-5, 10),
		},

		// wait for available index
		{
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
		// wait for all to be drained
		{
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

	output := newMockOutput(t)

	for _, tc := range testCases {
		if tc.before != nil {
			tc.before()
		}

		output.expect(tc.expectedResult)

		_, err := cache.Get(context.Background(), tc.from, tc.to, output)
		if err != nil && tc.expectedError != nil {
			assert.Equal(t, err.Error(), tc.expectedError.Error())
		}
	}
}
