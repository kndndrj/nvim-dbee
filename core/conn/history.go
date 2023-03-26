package conn

import (
	"encoding/gob"
	"errors"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"github.com/kndndrj/nvim-dbee/clients"
)

type historyRecord struct {
	file string
}

type historyMap struct {
	storage sync.Map
}

func (hm *historyMap) Store(key int, value historyRecord) {
	hm.storage.Store(key, value)
}

func (hm *historyMap) Load(key int) (historyRecord, bool) {
	val, ok := hm.storage.Load(key)
	if !ok {
		return historyRecord{}, false
	}

	return val.(historyRecord), true
}

func (hm *historyMap) Delete(key int) {
	hm.storage.Delete(key)
}

func (hm *historyMap) Keys() []int {
	var keys []int
	hm.storage.Range(func(key, value any) bool {
		k := key.(int)
		keys = append(keys, k)
		return true
	})

	return keys
}

type HistoryOutput struct {
	records historyMap
	last_id int
}

func NewHistory() *HistoryOutput {

	return &HistoryOutput{
		records: historyMap{},
		last_id: 0,
	}
}

// Act as an output (create a new record every time Write gets invoked)
func (o *HistoryOutput) Write(result Result) error {

	o.last_id++
	id := o.last_id

	fileName := "/tmp/" + uuid.New().String() + ".gob"

	// create a file
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// serialize the data
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(result)
	if err != nil {
		return err
	}

	rec := historyRecord{
		file: fileName,
	}
	o.records.Store(id, rec)

	return nil
}

// History is also a client
func (h *HistoryOutput) Execute(query string) (clients.Rows, error) {
	var result Result

	id, err := strconv.Atoi(query)
	if err != nil {
		return nil, err
	}

	rec, ok := h.records.Load(id)
	if !ok {
		return nil, errors.New("no such input in history")
	}
	fileName := rec.file

	// open the file
	file, err := os.Open(fileName)
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

func (h *HistoryOutput) Close() {
}

func (h *HistoryOutput) Schema() (clients.Schema, error) {
	return nil, nil
}

func (h *HistoryOutput) List() []string {
	keys := h.records.Keys()

	// sort the slice
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var strKeys []string
	for _, k := range keys {
		strKeys = append(strKeys, strconv.Itoa(k))
	}

	return strKeys
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
