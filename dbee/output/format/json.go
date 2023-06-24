package format

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/kndndrj/nvim-dbee/dbee/output"
)

var _ output.Formatter = (*JSON)(nil)

type JSON struct{}

func NewJSON() *JSON {
	return &JSON{}
}

func (jf *JSON) Name() string {
	return "json"
}

func (jf *JSON) parseSchemaFul(result models.Result) []map[string]any {
	var data []map[string]any

	for _, row := range result.Rows {

		record := make(map[string]any, len(row))
		for i, val := range row {
			var h string
			if i < len(result.Header) {
				h = result.Header[i]
			} else {
				h = fmt.Sprintf("<unknown-field-%d>", i)
			}
			record[h] = val
		}
		data = append(data, record)
	}
	return data
}

func (jf *JSON) parseSchemaLess(result models.Result) []any {
	var data []any

	for _, row := range result.Rows {
		if len(row) == 1 {
			data = append(data, row[0])
		} else if len(row) > 1 {
			data = append(data, row)
		}
	}
	return data
}

func (jf *JSON) Format(result models.Result, writer io.Writer) error {
	var data any
	switch result.Meta.SchemaType {
	case models.SchemaFul:
		data = jf.parseSchemaFul(result)
	case models.SchemaLess:
		data = jf.parseSchemaLess(result)
	}

	encoder := json.NewEncoder(writer)
	err := encoder.Encode(data)
	if err != nil {
		return err
	}
	return nil
}
