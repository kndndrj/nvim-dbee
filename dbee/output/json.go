package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

type JSONOutput struct {
	fileName string
	log      models.Logger
}

func NewJSONOutput(fileName string, logger models.Logger) *JSONOutput {
	return &JSONOutput{
		fileName: fileName,
		log:      logger,
	}
}

func (jo *JSONOutput) parseSchemaFul(result models.Result) []map[string]any {
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

func (jo *JSONOutput) parseSchemaLess(result models.Result) []any {
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

func (jo *JSONOutput) Write(result models.Result) error {
	file, err := os.Create(jo.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	var data any

	switch result.Meta.SchemaType {
	case models.SchemaFul:
		data = jo.parseSchemaFul(result)
	case models.SchemaLess:
		data = jo.parseSchemaLess(result)
	}

	encoder := json.NewEncoder(file)
	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	jo.log.Info("successfully saved json to " + jo.fileName)
	return nil
}
