package builders

import (
	"errors"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// ColumnsFromResultStream converts the result stream to columns.
// A result stream should return rows that are at least 2 columns wide and
// have the following structure:
//
//	1st elem: name - string
//	2nd elem: type - string
func ColumnsFromResultStream(rows core.ResultStream) ([]*core.Column, error) {
	var out []*core.Column

	for rows.HasNext() {
		row, err := rows.Next()
		if err != nil {
			return nil, fmt.Errorf("result.Next: %w", err)
		}

		if len(row) < 2 {
			return nil, errors.New("could not retrieve column info: insufficient data")
		}

		name, ok := row[0].(string)
		if !ok {
			return nil, errors.New("could not retrieve column info: name not a string")
		}

		typ, ok := row[1].(string)
		if !ok {
			return nil, errors.New("could not retrieve column info: type not a string")
		}

		column := &core.Column{
			Name: name,
			Type: typ,
		}

		out = append(out, column)
	}

	return out, nil
}
