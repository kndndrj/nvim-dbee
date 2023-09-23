package core

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"gotest.tools/assert"
)

type mockedResultStream struct {
	max     int
	current int
	sleep   time.Duration
}

func newMockedResultStream(maxRows int, sleep time.Duration) *mockedResultStream {
	return &mockedResultStream{
		max:   maxRows,
		sleep: sleep,
	}
}

func (mir *mockedResultStream) Meta() *Meta {
	return &Meta{}
}

func (mir *mockedResultStream) Header() Header {
	return Header{"header1", "header2"}
}

func (mir *mockedResultStream) Next() (Row, error) {
	if mir.current < mir.max {

		// sleep between iterations
		time.Sleep(mir.sleep)

		num := mir.current
		mir.current += 1
		return Row{num, strconv.Itoa(num)}, nil
	}

	return nil, errors.New("no next row")
}

func (mir *mockedResultStream) HasNext() bool {
	return mir.current < mir.max
}

func (mir *mockedResultStream) Close() {}

func (mir *mockedResultStream) Range(from int, to int) []Row {
	var rows []Row

	for i := from; i < to; i++ {
		rows = append(rows, Row{i, strconv.Itoa(i)})
	}
	return rows
}

func TestCache(t *testing.T) {
	// prepare cache and mocks
	result := new(Result)

	numOfRows := 10
	stream := newMockedResultStream(numOfRows, 0)

	err := result.setIter(stream)
	assert.NilError(t, err)

	type testCase struct {
		name          string
		from          int
		to            int
		before        func()
		expectedRows  []Row
		expectedError error
	}

	testCases := []testCase{
		{
			name:          "get all",
			from:          0,
			to:            -1,
			expectedRows:  stream.Range(0, numOfRows),
			expectedError: nil,
		},
		{
			name:          "get basic range",
			from:          0,
			to:            3,
			expectedRows:  stream.Range(0, 3),
			expectedError: nil,
		},
		{
			name:          "get last 2",
			from:          -3,
			to:            -1,
			expectedRows:  stream.Range(numOfRows-2, numOfRows),
			expectedError: nil,
		},
		{
			name:          "get only one",
			from:          0,
			to:            1,
			expectedRows:  stream.Range(0, 1),
			expectedError: nil,
		},

		{
			name:          "invalid range",
			from:          5,
			to:            1,
			expectedRows:  nil,
			expectedError: ErrInvalidRange(5, 1),
		},
		{
			name:          "invalid range (even if 10 can be higher than -1, its undefined and should fail)",
			from:          -5,
			to:            10,
			expectedRows:  nil,
			expectedError: ErrInvalidRange(-5, 10),
		},

		{
			name:          "wait for available index",
			from:          0,
			to:            3,
			expectedRows:  stream.Range(0, 3),
			expectedError: nil,
			before: func() {
				result.Wipe()
				// reset result with sleep between iterations
				err = result.setIter(newMockedResultStream(numOfRows, 500*time.Millisecond))
				assert.NilError(t, err)
			},
		},
		{
			name:          "wait for all to be drained",
			from:          0,
			to:            -1,
			expectedRows:  stream.Range(0, numOfRows),
			expectedError: nil,
			before: func() {
				result.Wipe()
				// reset result with sleep between iterations
				err = result.setIter(newMockedResultStream(numOfRows, 500*time.Millisecond))
				assert.NilError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.before != nil {
				tc.before()
			}

			rows, err := result.Rows(tc.from, tc.to)
			if err != nil && tc.expectedError != nil {
				assert.Equal(t, err.Error(), tc.expectedError.Error())
				return
			}

			assert.DeepEqual(t, rows, tc.expectedRows)
		})
	}
}
