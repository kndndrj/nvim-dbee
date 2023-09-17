package cache

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

func init() {
	// gob doesn't know how to encode/decode time otherwise
	gob.Register(time.Time{})
}

const archiveBasePath = "/tmp/dbee-history/"

// these variables create a file name for a specified type
var (
	archiveDir = func(callID string) string {
		return filepath.Join(archiveBasePath, callID)
	}

	callFile = func(callID string) string {
		return filepath.Join(archiveDir(callID), "call.gob")
	}
	metaFile = func(callID string) string {
		return filepath.Join(archiveDir(callID), "meta.gob")
	}
	headerFile = func(callID string) string {
		return filepath.Join(archiveDir(callID), "header.gob")
	}
	rowFile = func(callID string, i int) string {
		return filepath.Join(archiveDir(callID), fmt.Sprintf("row_%d.gob", i))
	}
)

// archive stores the cache record to disk as a set of gob files
func (c *cache) archive(record *cacheRecord) error {
	call := record.call
	result := record.result
	id := call.GetDetails().ID

	// create the directory for the history record
	err := os.MkdirAll(archiveDir(id), os.ModePerm)
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
	file, err := os.Create(headerFile(id))
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(result.header)
	if err != nil {
		return err
	}

	// meta
	file, err = os.Create(metaFile(id))
	if err != nil {
		return err
	}
	defer file.Close()

	encoder = gob.NewEncoder(file)
	err = encoder.Encode(*result.meta)
	if err != nil {
		return err
	}

	// rows
	chunkSize := 500
	length := len(result.rows)

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
			chunk := result.rows[chunkStart:chunkEnd]
			if len(chunk) == 0 {
				return nil
			}

			file, err := os.Create(rowFile(id, i))
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

	return nil
}

func (c *cache) archiveCall(call *call.Call) error {
	file, err := os.Create(callFile(call.GetDetails().ID))
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(*call.GetDetails())
	if err != nil {
		return err
	}

	return nil
}

// unarchive loads data from archive and starts filling cache
func (c *cache) unarchive(id string) error {
	rows, err := newArchiveRows(c.historyDir)
	if err != nil {
		return err
	}

	err = c.setResult(id, rows)
	if err != nil {
		return err
	}

	return nil
}

type archiveRows struct {
	header  models.Header
	meta    *models.Meta
	iter    func() (models.Row, error)
	hasNext func() bool
}

func newArchiveRows(callID string) (*archiveRows, error) {
	// read header and metadata
	header, meta, err := readHeaderAndMeta(dir)
	if err != nil {
		return nil, err
	}

	r := &archiveRows{
		header: header,
		meta:   meta,
	}

	// open the first file if it exists,
	// loop through its contents and try the next file
	fileExists := func(rowIndex int) bool {
		fileName := filepath.Join(dir, rowFile(rowIndex))
		_, err = os.Stat(fileName)
		return err == nil
	}

	// nextFile returns the contents of the next rows file
	index := 0
	nextFile := func() (resultRows []models.Row, err error, isLast bool) {
		file, err := os.Open(filepath.Join(dir, rowFile(index)))
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
		return rows, nil, !fileExists(index + 1)
	}

	// holds rows from current file in memory
	currentRows := []models.Row{}
	maxIndex := -1
	isLastFile := false
	hasNext := true
	i := 0
	r.iter = func() (models.Row, error) {
		if i == maxIndex && isLastFile {
			hasNext = false
		}
		if i > maxIndex {
			if isLastFile {
				return nil, errors.New("no next row")
			}

			var err error
			currentRows, err, isLastFile = nextFile()
			if err != nil {
				return nil, err
			}
			maxIndex = len(currentRows) - 1
			i = 0
		}
		val := currentRows[i]
		i++
		return val, nil
	}

	r.hasNext = func() bool {
		return hasNext
	}

	return r, nil
}

func (r *archiveRows) readHeaderAndMeta(dir string) (models.Header, *models.Meta, error) {
	// header
	var header models.Header
	fileName := filepath.Join(dir, headerFilename)
	file, err := os.Open(fileName)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&header)
	if err != nil {
		return nil, nil, err
	}

	// meta
	var meta models.Meta
	fileName = filepath.Join(dir, metaFilename)
	file, err = os.Open(fileName)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	decoder = gob.NewDecoder(file)
	err = decoder.Decode(&meta)
	if err != nil {
		return nil, nil, err
	}

	return header, &meta, nil
}

func (r *archiveRows) Meta() *models.Meta {
	return r.meta
}

func (r *archiveRows) Header() models.Header {
	return r.header
}

func (r *archiveRows) Next() (models.Row, error) {
	return r.iter()
}

func (r *archiveRows) HasNext() bool {
	return r.hasNext()
}

func (r *archiveRows) Close() {
	// no-op
}
