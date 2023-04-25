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

func (jo *JSONOutput) Write(result models.Result) error {
	file, err := os.Create(jo.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	var data []map[string]string
	header := result.Header
	lh := len(header)
	for _, row := range result.Rows {
		rec := make(map[string]string)
		for i, r := range row {
			h := ""
			if i < lh {
				h = header[i]
			}
			rec[h] = fmt.Sprint(r)
		}
		data = append(data, rec)
	}

	encoder := json.NewEncoder(file) 
	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	jo.log.Info("successfully saved json to " + jo.fileName)
	return nil
}
