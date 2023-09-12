package vim

import (
	"testing"

	"gotest.tools/assert"
)

func TestFuncArgs_Parse(t *testing.T) {
	type fields struct {
		StringField  string `arg:"some_name"`
		IntField     int    `arg:",optional"`
		BoolField    bool   `arg:"some_bool,optional"`
		Int64Field   int64
		Float64Field float64 `arg:"some_float,optional"`
	}

	type testCase struct {
		name           string
		raw            map[string]any
		expectedResult *fields
		expectedError  error
	}

	// structs aren't supported
	t.Run("Type param not a struct", func(t *testing.T) {
		funcArgs := FuncArgs[int]{}

		_, err := funcArgs.Parse()
		assert.ErrorContains(t, err, ErrNotAStruct(0).Error())
	})

	testCases := []testCase{
		{
			name: "Basic Parse",
			raw: map[string]any{
				"some_name":  "name",
				"IntField":   3,
				"some_bool":  true,
				"Int64Field": int64(23),
				"some_float": float64(2.3),
			},
			expectedResult: &fields{
				StringField:  "name",
				IntField:     3,
				BoolField:    true,
				Int64Field:   int64(23),
				Float64Field: float64(2.3),
			},
			expectedError: nil,
		},
		{
			name: "Parse uint64 as int",
			raw: map[string]any{
				"some_name":  "name",
				"IntField":   uint64(23),
				"Int64Field": int64(23),
			},
			expectedResult: &fields{
				StringField: "name",
				IntField:    23,
				Int64Field:  int64(23),
			},
			expectedError: nil,
		},
		{
			name: "Required field not set",
			raw: map[string]any{
				"Int64Field": int64(23),
			},
			expectedResult: nil,
			expectedError:  ErrRequiredFieldNotSet("some_name"),
		},
		{
			name: "Invalid field type",
			raw: map[string]any{
				"some_name":  3,
				"Int64Field": int64(23),
			},
			expectedResult: nil,
			expectedError:  ErrInvalidFieldType("some_name", "", 0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			funcArgs := FuncArgs[fields]{}
			funcArgs.Set(tc.raw)

			parsed, err := funcArgs.Parse()
			if err != nil {
				assert.ErrorContains(t, err, tc.expectedError.Error())
				return
			}

			assert.DeepEqual(t, tc.expectedResult, parsed)
		})
	}
}
