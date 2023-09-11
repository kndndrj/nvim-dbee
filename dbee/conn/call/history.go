package call

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

const (
	metaFilename    = "meta.gob"
	headerFilename  = "header.gob"
	historyBasePath = "/tmp/dbee-history/"
)

var rowsFilename = func(i int) string { return fmt.Sprintf("row_%d.gob", i) }

var callHistoryPath = func(callID string) string { return filepath.Join(historyBasePath, callID) }

func init() {
	// gob doesn't know how to encode/decode time otherwise
	gob.Register(time.Time{})
}

var ErrHistoryAlreadyFilled = errors.New("history is already filled")

// archive stores result to disk as a set of gob files
func (c *cache) archive(result models.Result) error {
	if c.historyState != CacheStateEmpty {
		return ErrHistoryAlreadyFilled
	}

	c.historyState = CacheStateFilling

	// create the directory for the history record
	err := os.MkdirAll(c.historyDir, os.ModePerm)
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
	fileName := filepath.Join(c.historyDir, headerFilename)
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
	fileName = filepath.Join(c.historyDir, metaFilename)
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
		i := i
		g.Go(func() error {
			// get chunk
			chunkStart := chunkSize * i
			chunkEnd := chunkSize * (i + 1)
			if chunkEnd > length {
				chunkEnd = length
			}
			chunk := result.Rows[chunkStart:chunkEnd]
			if len(chunk) == 0 {
				return nil
			}

			fileName := filepath.Join(c.historyDir, rowsFilename(i))
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

	c.historyState = CacheStateFilled

	return nil
}

// unarchive loads data from archive and starts filling cache
func (c *cache) unarchive(ctx context.Context) error {
	rows, err := newHistoryRows((c.historyDir))
	if err != nil {
		return err
	}

	err = c.Set(ctx, rows)
	if err != nil {
		return err
	}

	return nil
}

type historyRows struct {
	header models.Header
	meta   models.Meta
	iter   func() (models.Row, error)
}

func newHistoryRows(dir string) (*historyRows, error) {
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
		fileName := filepath.Join(dir, rowsFilename(index))
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
	maxIndex := -1
	i := 0
	iter := func() (models.Row, error) {
		if i > maxIndex {
			var last bool
			var err error
			currentRows, err, last = nextFile()
			if err != nil {
				return nil, err
			}
			if last {
				return nil, nil
			}
			maxIndex = len(currentRows) - 1
			i = 0
		}
		val := currentRows[i]
		i++
		return val, nil
	}

	return &historyRows{
		header: header,
		meta:   meta,
		iter:   iter,
	}, nil
}

func readHeaderAndMeta(dir string) (models.Header, models.Meta, error) {
	// header
	var header models.Header
	fileName := filepath.Join(dir, headerFilename)
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
	fileName = filepath.Join(dir, metaFilename)
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

func (r *historyRows) Meta() (models.Meta, error) {
	return r.meta, nil
}

func (r *historyRows) Header() (models.Header, error) {
	return r.header, nil
}

func (r *historyRows) Next() (models.Row, error) {
	return r.iter()
}

func (r *historyRows) Close() {
	// no-op
}
