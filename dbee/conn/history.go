package conn

import (
	"context"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

func init() {
	// gob doesn't know how to encode/decode time otherwise
	gob.Register(time.Time{})
}

type HistoryState int

const (
	HistoryStateAvailable HistoryState = iota
	HistoryStateFilling
	HistoryStateFilled
	HistoryStateFailed
)

// HistoryOutput is a subcomponent of a call, which holds the result on disk
type HistoryOutput struct {
	directory string
	state     HistoryState
}

func NewHistory(searchID string) *HistoryOutput {
	ho := &HistoryOutput{
		// TODO: handle windows
		directory: filepath.Join("/tmp/dbee-history", searchID),
		state:     HistoryStateAvailable,
	}

	_, err := os.Stat(ho.directory)
	if err == nil {
		ho.state = HistoryStateFilled
	}
	return ho
}

func (ho *HistoryOutput) HasResult() bool {
	return ho.state == HistoryStateFilled
}

// Act as an output (create a new record every time Write gets invoked)
func (ho *HistoryOutput) Write(_ context.Context, result models.Result) error {
	if ho.state != HistoryStateAvailable {
		return ErrAlreadyFilled
	}

	ho.state = HistoryStateFilling

	// create the directory for the history record
	err := os.MkdirAll(ho.directory, os.ModePerm)
	if err != nil {
		return err
	}

	// serialize the data
	// files inside the directory ..../call_id/:
	// header.gob - header
	// meta.gob - meta
	// row_0.gob - first row
	// row_n.gob - n-th row

	// header
	fileName := filepath.Join(ho.directory, "header.gob")
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
	fileName = filepath.Join(ho.directory, "meta.gob")
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

			fileName := filepath.Join(ho.directory, "row_"+strconv.Itoa(index)+".gob")
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

	ho.state = HistoryStateFilled

	return nil
}

func (ho *HistoryOutput) Query(_ context.Context) (models.IterResult, error) {
	if ho.state != HistoryStateFilled {
		return nil, errors.New("no result stored in history")
	}

	return newHistoryRows(ho.directory)
}

// scanOld scans the ho.directory/ho.searchId to find any existing history records
// func (ho *HistoryOutput) scanOld() error {
// 	// list directory contents
// 	searchDir := filepath.Join(ho.directory, ho.searchID)

// 	// check if dir exists and is a directory
// 	dirInfo, err := os.Stat(searchDir)
// 	if os.IsNotExist(err) || !dirInfo.IsDir() {
// 		return nil
// 	}

// 	contents, err := os.ReadDir(searchDir)
// 	if err != nil {
// 		return err
// 	}
// 	for _, c := range contents {
// 		if !c.IsDir() {
// 			continue
// 		}

// 		id := c.Name()

// 		dir := filepath.Join(searchDir, c.Name())

// 		// header
// 		var header models.Header
// 		fileName := filepath.Join(dir, "header.gob")
// 		file, err := os.Open(fileName)
// 		if err != nil {
// 			return err
// 		}
// 		defer file.Close()

// 		decoder := gob.NewDecoder(file)
// 		err = decoder.Decode(&header)
// 		if err != nil {
// 			return err
// 		}

// 		// meta
// 		var meta models.Meta
// 		fileName = filepath.Join(dir, "meta.gob")
// 		file, err = os.Open(fileName)
// 		if err != nil {
// 			return err
// 		}
// 		defer file.Close()

// 		decoder = gob.NewDecoder(file)
// 		err = decoder.Decode(&meta)
// 		if err != nil {
// 			return err
// 		}

// 		rec := historyRecord{
// 			dir:    dir,
// 			header: header,
// 			meta:   meta,
// 		}

// 		ho.records.store(id, rec)

// 	}

// 	return nil
// }

type HistoryRows struct {
	header models.Header
	meta   models.Meta
	iter   func() (models.Row, error)
}

func newHistoryRows(dir string) (*HistoryRows, error) {
	// read header and metadata
	header, meta, err := readHeaderAndMeta(dir)
	if err != nil {
		return nil, err
	}

	// open the first file if it exists,
	// loop through its contents and try the next file

	// nextFile returns the contents of the next rows file
	index := 0
	nextFile := func() ([]models.Row, error, bool) {
		fileName := filepath.Join(dir, "row_"+strconv.Itoa(index)+".gob")
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
		header: header,
		meta:   meta,
		iter:   iter,
	}, nil
}

func readHeaderAndMeta(dir string) (models.Header, models.Meta, error) {
	// header
	var header models.Header
	fileName := filepath.Join(dir, "header.gob")
	file, err := os.Open(fileName)
	if err != nil {
		return nil, models.Meta{}, err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&header)
	if err != nil {
		return nil, models.Meta{}, err
	}

	// meta
	var meta models.Meta
	fileName = filepath.Join(dir, "meta.gob")
	file, err = os.Open(fileName)
	if err != nil {
		return nil, models.Meta{}, err
	}
	defer file.Close()

	decoder = gob.NewDecoder(file)
	err = decoder.Decode(&meta)
	if err != nil {
		return nil, models.Meta{}, err
	}

	return header, meta, nil
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
	// no-op
}
