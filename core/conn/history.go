package conn

import (
	"encoding/gob"
	"errors"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/google/uuid"
)

type historyRecord struct {
	file string
}

type historyMap struct {
	storage sync.Map
}

func (hm *historyMap) store(key int, value historyRecord) {
	hm.storage.Store(key, value)
}

func (hm *historyMap) load(key int) (historyRecord, bool) {
	val, ok := hm.storage.Load(key)
	if !ok {
		return historyRecord{}, false
	}

	return val.(historyRecord), true
}

func (hm *historyMap) keys() []int {
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
	o.records.store(id, rec)

	return nil
}

// History is also a client
func (h *HistoryOutput) Query(historyId string) (IterResult, error) {
	var result Result

	id, err := strconv.Atoi(historyId)
	if err != nil {
		return nil, err
	}

	rec, ok := h.records.load(id)
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

func (h *HistoryOutput) Schema() (Schema, error) {
	return nil, nil
}

func (h *HistoryOutput) List() []string {
	keys := h.records.keys()

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
	iter   func() Row
	header Header
}

func newHistoryRows(result Result) *HistoryRows {
	iter := getIter(result)

	return &HistoryRows{
		iter:   iter,
		header: result.Header,
	}
}

func (r *HistoryRows) Header() (Header, error) {
	return r.header, nil
}

func getIter(result Result) func() Row {
	max := len(result.Rows) - 1
	i := 0
	return func() Row {
		if i > max {
			return nil
		}
		val := result.Rows[i]
		i++
		return val
	}
}

func (r *HistoryRows) Next() (Row, error) {
	return r.iter(), nil
}

func (r *HistoryRows) Close() {
}
