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

func (co *CSV) parseSchemaFul(result models.IterResult) ([][]string, error) {
	data := [][]string{
		result.Header(),
	}
	for result.HasNext() {

		row, err := result.Next()
		if err != nil {
			return nil, err
		}

		var csvRow []string
		for _, rec := range row {
			csvRow = append(csvRow, fmt.Sprint(rec))
		}
		data = append(data, csvRow)
	}

	return data, nil
}

func (cf *CSV) Format(result models.IterResult, writer io.Writer) error {
	// parse as if schema is defined regardles of schema presence in the result
	data, err := cf.parseSchemaFul(result)
	if err != nil {
		return err
	}

	w := csv.NewWriter(writer)
	err = w.WriteAll(data)
	if err != nil {
		return err
	}
	if err := w.Error(); err != nil {
		return err
	}
	return nil
}
