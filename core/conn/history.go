package conn

import (
	"encoding/gob"
	"os"
	"time"

	"github.com/kndndrj/nvim-dbee/clients"
)

type HistoryOutput struct {
	fileName string
}

func newHistory() *HistoryOutput {
	fileName := "/tmp/" + time.Now().Format("20060102150405") + ".gob"

	return &HistoryOutput{
		fileName: fileName,
	}
}

func (o *HistoryOutput) Write(result Result) error {
	// create a file
	file, err := os.Create(o.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// serialize the data
	encoder := gob.NewEncoder(file)
	return encoder.Encode(result)
}

// History is also a client
func (o *HistoryOutput) Execute(query string) (clients.Rows, error) {
	var result Result

	// open the file
	file, err := os.Open(o.fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&result)
	if err != nil {
		return nil, err
	}

	rows := newHistoryRows(result)

	return rows, nil
}

func (o *HistoryOutput) Close() {
}

func (o *HistoryOutput) Schema() (clients.Schema, error) {
	return nil, nil
}

type HistoryRows struct {
	iter   func() clients.Row
	header clients.Header
}

func newHistoryRows(result Result) *HistoryRows {
	iter := getIter(result)

	return &HistoryRows{
		iter:   iter,
		header: result.Header,
	}
}

func (r *HistoryRows) Header() (clients.Header, error) {
	return r.header, nil
}

func getIter(result Result) func() clients.Row {
	max := len(result.Rows) - 1
	i := 0
	return func() clients.Row {
		if i > max {
			return nil
		}
		val := result.Rows[i]
		i++
		return val
	}
}

func (r *HistoryRows) Next() (clients.Row, error) {
	return r.iter(), nil
}

func (r *HistoryRows) Close() {
}
