package format

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

var _ core.Formatter = (*CSV)(nil)

type CSV struct{}

func NewCSV() *CSV {
	return &CSV{}
}

func (cf *CSV) parseSchemaFul(header core.Header, rows []core.Row) [][]string {
	data := [][]string{
		header,
	}
	for _, row := range rows {
		var csvRow []string
		for _, rec := range row {
			csvRow = append(csvRow, fmt.Sprint(rec))
		}
		data = append(data, csvRow)
	}

	return data
}

func (cf *CSV) Format(header core.Header, rows []core.Row, _ *core.FormatterOptions) ([]byte, error) {
	// parse as if schema is defined regardles of schema presence in the result
	data := cf.parseSchemaFul(header, rows)

	b := new(bytes.Buffer)
	w := csv.NewWriter(b)

	err := w.WriteAll(data)
	if err != nil {
		return nil, fmt.Errorf("w.WriteAll: %w", err)
	}

	return b.Bytes(), nil
}
