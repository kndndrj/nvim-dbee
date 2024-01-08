package format

import (
	"encoding/json"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

var _ core.Formatter = (*JSON)(nil)

type JSON struct{}

func NewJSON() *JSON {
	return &JSON{}
}

func (jf *JSON) parseSchemaFul(header core.Header, rows []core.Row) []map[string]any {
	var data []map[string]any

	for _, row := range rows {
		record := make(map[string]any, len(row))
		for i, val := range row {
			var h string
			if i < len(header) {
				h = header[i]
			} else {
				h = fmt.Sprintf("<unknown-field-%d>", i)
			}
			record[h] = val
		}
		data = append(data, record)
	}

	return data
}

func (jf *JSON) parseSchemaLess(header core.Header, rows []core.Row) []any {
	var data []any

	for _, row := range rows {
		if len(row) == 1 {
			data = append(data, row[0])
		} else if len(row) > 1 {
			data = append(data, row)
		}
	}
	return data
}

func (jf *JSON) Format(header core.Header, rows []core.Row, opts *core.FormatterOptions) ([]byte, error) {
	var data any
	switch opts.SchemaType {
	case core.SchemaLess:
		data = jf.parseSchemaLess(header, rows)
	case core.SchemaFul:
		fallthrough
	default:
		data = jf.parseSchemaFul(header, rows)
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("json.MarshalIndent: %w", err)
	}

	return out, nil
}
