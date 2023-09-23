package format

import (
	"encoding/json"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var _ call.Formatter = (*JSON)(nil)

type JSON struct{}

func NewJSON() *JSON {
	return &JSON{}
}

func (jf *JSON) Name() string {
	return "json"
}

func (jf *JSON) parseSchemaFul(header models.Header, rows []models.Row) []map[string]any {
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

func (jf *JSON) parseSchemaLess(header models.Header, rows []models.Row) []any {
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

func (jf *JSON) Format(header models.Header, rows []models.Row, opts *models.FormatOpts) ([]byte, error) {
	var data any
	switch opts.SchemaType {
	case models.SchemaLess:
		data = jf.parseSchemaLess(header, rows)
	case models.SchemaFul:
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
