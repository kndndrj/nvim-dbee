package output

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"github.com/kndndrj/nvim-dbee/clients"
)

type HistoryOutput struct {
	fileName string
	header   clients.Header
	rows     []clients.Row
}

func NewHistoryOutput(id string) *HistoryOutput {
	fileName := "/tmp/" + id + time.Now().Format("20060102150405") + ".gob"

	return &HistoryOutput{
		fileName: fileName,
	}
}

func (o *HistoryOutput) AddHeader(header clients.Header) error {
	o.header = header
	return nil
}

func (o *HistoryOutput) AddRow(row clients.Row) error {
	o.rows = append(o.rows, row)
	return nil
}

func (o *HistoryOutput) Flush() error {
	file, err := os.Create(o.fileName)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			// TODO: add propper logging
			fmt.Println("Failed to close the history file" + o.fileName)
		}
	}()

	// serialize the data
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(o.rows)
	return err
}

// History is also a client
func (o *HistoryOutput) Execute(query string) (clients.Rows, error) {
	return nil, nil
}

func (o *HistoryOutput) Close() {
}

func (o *HistoryOutput) Schema() (clients.Schema, error) {
	return nil, nil
}
