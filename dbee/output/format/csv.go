package format

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/kndndrj/nvim-dbee/dbee/models"
	"github.com/kndndrj/nvim-dbee/dbee/output"
)

var _ output.Formatter = (*CSV)(nil)

type CSV struct{}

func NewCSV() *CSV {
	return &CSV{}
}

func (cf *CSV) Name() string {
	return "csv"
}

func (co *CSV) parseSchemaFul(result models.Result) [][]string {
	data := [][]string{
		result.Header,
	}
	for _, row := range result.Rows {
		var csvRow []string
		for _, rec := range row {
			csvRow = append(csvRow, fmt.Sprint(rec))
		}
		data = append(data, csvRow)
	}

	return data
}

func (cf *CSV) Format(result models.Result, writer io.Writer) error {
	// parse as if schema is defined regardles of schema presence in the result
	data := cf.parseSchemaFul(result)

	w := csv.NewWriter(writer)
	err := w.WriteAll(data)
	if err != nil {
		return err
	}
	if err := w.Error(); err != nil {
		return err
	}
	return nil
}
