package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExpand(t *testing.T) {
	r := require.New(t)

	testCases := []struct {
		input    string
		expected string
	}{
		{"normal string", "normal string"},
		{"{{ env `HOME` }}", os.Getenv("HOME")},
		{"{{ exec `echo \"hello\nbuddy\" | grep buddy` }}", "buddy"},
	}

	for _, tc := range testCases {
		actual, err := expand(tc.input)
		r.NoError(err)

		r.Equal(tc.expected, actual)
	}
}
