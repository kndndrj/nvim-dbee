package format

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var _ call.Formatter = (*CSV)(nil)

type CSV struct{}

func NewCSV() *CSV {
	return &CSV{}
}

func (cf *CSV) Name() string {
	return "csv"
}

func (cf *CSV) parseSchemaFul(header models.Header, rows []models.Row) [][]string {
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

func (cf *CSV) Format(header models.Header, rows []models.Row, _ *models.FormatOpts) ([]byte, error) {
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
