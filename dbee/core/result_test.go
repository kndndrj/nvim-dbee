package core_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/mock"
)

func TestResult(t *testing.T) {
	type testCase struct {
		name          string
		from          int
		to            int
		input         []core.Row
		expected      []core.Row
		expectedError error
	}

	testCases := []testCase{
		{
			name:          "get all",
			from:          0,
			to:            -1,
			input:         mock.NewRows(0, 10),
			expected:      mock.NewRows(0, 10),
			expectedError: nil,
		},
		{
			name:          "get basic range",
			from:          0,
			to:            3,
			input:         mock.NewRows(0, 10),
			expected:      mock.NewRows(0, 3),
			expectedError: nil,
		},
		{
			name:          "get last 2",
			from:          -3,
			to:            -1,
			input:         mock.NewRows(0, 10),
			expected:      mock.NewRows(8, 10),
			expectedError: nil,
		},
		{
			name:          "get only one",
			from:          0,
			to:            1,
			input:         mock.NewRows(0, 10),
			expected:      mock.NewRows(0, 1),
			expectedError: nil,
		},

		{
			name:          "invalid range",
			from:          5,
			to:            1,
			input:         mock.NewRows(0, 10),
			expected:      nil,
			expectedError: core.ErrInvalidRange(5, 1),
		},
		{
			name:          "invalid range (even if 10 can be higher than -1, its undefined and should fail)",
			from:          -5,
			to:            10,
			input:         mock.NewRows(0, 10),
			expected:      nil,
			expectedError: core.ErrInvalidRange(-5, 10),
		},

		{
			name:          "wait for available index",
			from:          0,
			to:            3,
			input:         mock.NewRows(0, 10),
			expected:      mock.NewRows(0, 3),
			expectedError: nil,
		},
		{
			name:          "wait for all to be drained",
			from:          0,
			to:            -1,
			input:         mock.NewRows(0, 10),
			expected:      mock.NewRows(0, 10),
			expectedError: nil,
		},
	}

	result := new(core.Result)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			// wipe any previous result
			result.Wipe()

			// set a new iterator with input
			err := result.SetIter(mock.NewResultStream(tc.input, mock.ResultStreamWithNextSleep(300*time.Millisecond)), nil)
			r.NoError(err)

			rows, err := result.Rows(tc.from, tc.to)
			if err != nil {
				r.ErrorContains(tc.expectedError, err.Error())
			}
			r.Equal(rows, tc.expected)
		})
	}
}
