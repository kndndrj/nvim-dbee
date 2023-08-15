package conn

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
	"golang.org/x/sync/errgroup"
)

type historyRecord struct {
	dir    string
	header models.Header
	meta   models.Meta
}

// key int64 - unix timestamp
// value historyRecord
type historyMap struct {
	storage sync.Map
}

func (hm *historyMap) store(key int64, value historyRecord) {
	hm.storage.Store(key, value)
}

func (hm *historyMap) load(key int64) (historyRecord, bool) {
	val, ok := hm.storage.Load(key)
	if !ok {
		return historyRecord{}, false
	}

	return val.(historyRecord), true
}

func (hm *historyMap) keys() []int64 {
	var keys []int64
	hm.storage.Range(func(key, value any) bool {
		k := key.(int64)
		keys = append(keys, k)
		return true
	})

	return keys
}

type HistoryOutput struct {
	records historyMap
	// searchId is used to identify history records over restarts
	searchId  string
	directory string
	log       models.Logger
}

func NewHistory(searchId string, logger models.Logger) *HistoryOutput {
	// gob doesn't know how to encode/decode time otherwise
	gob.Register(time.Time{})

	h := &HistoryOutput{
		records:  historyMap{},
		searchId: searchId,
		// TODO: handle windows
		directory: "/tmp/dbee-history",
		log:       logger,
	}

	// concurrently gather info about any existing histories
	go func() {
		err := h.scanOld()
		if err != nil {
			logger.Error(err.Error())
		}
	}()

	return h
}

// Act as an output (create a new record every time Write gets invoked)
func (ho *HistoryOutput) Write(result models.Result) error {
	// use unix nanoseconds as an id - easier sorting over restarts
	id := time.Now().UnixNano()

	// someting like /tmp/dbee/conn_id/unix_timestamp/
	dir := filepath.Join(ho.directory, ho.searchId, strconv.FormatInt(id, 10))

	// create the directory for the history record
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	// serialize the data
	// files inside the directory ..../unix_timestamp/:
	// header.gob - header
	// meta.gob - meta
	// row_0.gob - first row
	// row_n.gob - n-th row

	// header
	fileName := filepath.Join(dir, "header.gob")
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(result.Header)
	if err != nil {
		return err
	}

	// meta
	fileName = filepath.Join(dir, "meta.gob")
	file, err = os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder = gob.NewEncoder(file)
	err = encoder.Encode(result.Meta)
	if err != nil {
		return err
	}

	// rows
	chunkSize := 500
	length := len(result.Rows)

	// write chunks concurrently
	g := &errgroup.Group{}
	g.SetLimit(10)
	for i := 0; i <= length/chunkSize; i++ {
		index := i
		g.Go(func() error {
			// get chunk
			chunkStart := chunkSize * index
			chunkEnd := chunkSize * (index + 1)
			if chunkEnd > length {
				chunkEnd = length
			}
			chunk := result.Rows[chunkStart:chunkEnd]
			if len(chunk) == 0 {
				return nil
			}

			fileName := filepath.Join(dir, "row_"+strconv.Itoa(index)+".gob")
			file, err := os.Create(fileName)
			if err != nil {
				return err
			}
			defer file.Close()

			encoder := gob.NewEncoder(file)
			err = encoder.Encode(chunk)
			if err != nil {
				return err
			}

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	rec := historyRecord{
		dir:    dir,
		header: result.Header,
		meta:   result.Meta,
	}
	ho.records.store(id, rec)

	return nil
}

// History is also a client
func (ho *HistoryOutput) Query(historyId string) (models.IterResult, error) {
	i, err := strconv.Atoi(historyId)
	if err != nil {
		return nil, err
	}
	id := int64(i)

	rec, ok := ho.records.load(id)
	if !ok {
		return nil, errors.New("no such input in history")
	}

	return newHistoryRows(rec)
}

func (ho *HistoryOutput) Layout() ([]models.Layout, error) {
	keys := ho.records.keys()

	// sort the slice
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var layouts []models.Layout
	for _, key := range keys {

		rec, ok := ho.records.load(key)
		if !ok {
			continue
		}

		layout := models.Layout{
			Name:     strconv.Itoa(int(key)),
			Schema:   "",
			Database: "",
			Type:     models.LayoutTypeHistory,
			Children: []models.Layout{
				{
					Name:     rec.meta.Timestamp.String(),
					Schema:   "",
					Database: "",
					Type:     models.LayoutTypeNone,
				},
				{
					Name:     rec.meta.Query,
					Schema:   "",
					Database: "",
					Type:     models.LayoutTypeNone,
				},
			},
		}
		layouts = append(layouts, layout)
	}

	return layouts, nil
}

// scanOld scans the ho.directory/ho.searchId to find any existing history records
func (ho *HistoryOutput) scanOld() error {
	// list directory contents
	searchDir := filepath.Join(ho.directory, ho.searchId)

	// check if dir exists and is a directory
	dirInfo, err := os.Stat(searchDir)
	if os.IsNotExist(err) || !dirInfo.IsDir() {
		return nil
	}

	contents, err := os.ReadDir(searchDir)
	if err != nil {
		return err
	}
	for _, c := range contents {
		if !c.IsDir() {
			continue
		}

		i, err := strconv.Atoi(c.Name())
		if err != nil {
			return err
		}
		id := int64(i)

		dir := filepath.Join(searchDir, c.Name())

		// header
		var header models.Header
		fileName := filepath.Join(dir, "header.gob")
		file, err := os.Open(fileName)
		if err != nil {
			return err
		}
		defer file.Close()

		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&header)
		if err != nil {
			return err
		}

		// meta
		var meta models.Meta
		fileName = filepath.Join(dir, "meta.gob")
		file, err = os.Open(fileName)
		if err != nil {
			return err
		}
		defer file.Close()

		decoder = gob.NewDecoder(file)
		err = decoder.Decode(&meta)
		if err != nil {
			return err
		}

		rec := historyRecord{
			dir:    dir,
			header: header,
			meta:   meta,
		}

		ho.records.store(id, rec)

	}

	return nil
}

type HistoryRows struct {
	header models.Header
	meta   models.Meta
	iter   func() (models.Row, error)
}

func newHistoryRows(record historyRecord) (*HistoryRows, error) {
	// open the first file if it exists,
	// loop through its contents and try the next file

	// nextFile returns the contents of the next rows file
	index := 0
	nextFile := func() ([]models.Row, error, bool) {
		fileName := filepath.Join(record.dir, "row_"+strconv.Itoa(index)+".gob")
		_, err := os.Stat(fileName)
		if os.IsNotExist(err) {
			return nil, nil, true
		}
		if err != nil {
			return nil, err, false
		}

		file, err := os.Open(fileName)
		if err != nil {
			return nil, err, false
		}
		defer file.Close()

		var rows []models.Row

		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&rows)
		if err != nil {
			return nil, err, false
		}

		index++
		return rows, nil, false
	}

	// holds rows from current file in memory
	currentRows := []models.Row{}
	max := -1
	i := 0
	iter := func() (models.Row, error) {
		if i > max {
			var last bool
			var err error
			currentRows, err, last = nextFile()
			if err != nil {
				return nil, err
			}
			if last {
				return nil, nil
			}
			max = len(currentRows) - 1
			i = 0
		}
		val := currentRows[i]
		i++
		return val, nil
	}

	return &HistoryRows{
		header: record.header,
		meta:   record.meta,
		iter:   iter,
	}, nil
}

func (r *HistoryRows) Meta() (models.Meta, error) {
	return r.meta, nil
}

func (r *HistoryRows) Header() (models.Header, error) {
	return r.header, nil
}

func (r *HistoryRows) Next() (models.Row, error) {
	return r.iter()
}

func (r *HistoryRows) Close() {
}
