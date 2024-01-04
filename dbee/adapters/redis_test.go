package adapters

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRedisCmd(t *testing.T) {
	r := require.New(t)

	type testCase struct {
		unparsed       string
		expectedResult []any
		expectedError  error
	}

	testCases := []testCase{
		// these should work
		{
			unparsed:       `set key val`,
			expectedResult: []any{"set", "key", "val"},
			expectedError:  nil,
		},
		{
			unparsed:       `set key "double quoted val"`,
			expectedResult: []any{"set", "key", "double quoted val"},
			expectedError:  nil,
		},
		{
			unparsed:       `set key 'single quoted val'`,
			expectedResult: []any{"set", "key", "single quoted val"},
			expectedError:  nil,
		},
		{
			unparsed:       `set key 'single quoted val with nested unescaped double quote (")'`,
			expectedResult: []any{"set", "key", "single quoted val with nested unescaped double quote (\")"},
			expectedError:  nil,
		},
		{
			unparsed:       `set key 'single quoted val with nested escaped double quote (\")'`,
			expectedResult: []any{"set", "key", "single quoted val with nested escaped double quote (\")"},
			expectedError:  nil,
		},
		{
			unparsed:       `set key 'single quoted val with nested escaped single quote (\')'`,
			expectedResult: []any{"set", "key", "single quoted val with nested escaped single quote (')"},
			expectedError:  nil,
		},
		{
			unparsed:       `set key "double quoted val with nested unescaped single quote (')"`,
			expectedResult: []any{"set", "key", "double quoted val with nested unescaped single quote (')"},
			expectedError:  nil,
		},
		{
			unparsed:       `set key "double quoted val with nested escaped single quote (\')"`,
			expectedResult: []any{"set", "key", "double quoted val with nested escaped single quote (')"},
			expectedError:  nil,
		},
		{
			unparsed:       `set key "double quoted val with nested escaped double quote (\")"`,
			expectedResult: []any{"set", "key", "double quoted val with nested escaped double quote (\")"},
			expectedError:  nil,
		},

		// these shouldn't work
		{
			unparsed:       `set key "unmatched double quoted val`,
			expectedResult: nil,
			expectedError:  ErrUnmatchedDoubleQuote(9),
		},
		{
			unparsed:       `set key 'unmatched single quoted val`,
			expectedResult: nil,
			expectedError:  ErrUnmatchedSingleQuote(9),
		},
		{
			unparsed:       `set key "double quoted val with nested unescaped double quote (")"`,
			expectedResult: nil,
			expectedError:  ErrUnmatchedDoubleQuote(64),
		},
		{
			unparsed:       `set key 'single quoted val with nested unescaped single quote (')'`,
			expectedResult: nil,
			expectedError:  ErrUnmatchedSingleQuote(64),
		},
	}

	for _, tc := range testCases {
		parsed, err := parseRedisCmd(tc.unparsed)
		if err != nil {
			r.Equal(err.Error(), tc.expectedError.Error())
			continue
		}
		r.Equal(parsed, tc.expectedResult)
	}
}
