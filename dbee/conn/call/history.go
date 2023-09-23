package call

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
	"golang.org/x/sync/errgroup"
)

func init() {
	// gob doesn't know how to encode/decode time otherwise
	gob.Register(time.Time{})
}

const archiveBasePath = "/tmp/dbee-history/"

// these variables create a file name for a specified type
var (
	archiveDir = func(callID StatID) string {
		return filepath.Join(archiveBasePath, string(callID))
	}

	metaFile = func(callID StatID) string {
		return filepath.Join(archiveDir(callID), "meta.gob")
	}
	headerFile = func(callID StatID) string {
		return filepath.Join(archiveDir(callID), "header.gob")
	}
	rowFile = func(callID StatID, i int) string {
		return filepath.Join(archiveDir(callID), fmt.Sprintf("row_%d.gob", i))
	}
)

type archive struct {
	id       StatID
	isFilled bool
}

func newArchive(id StatID) *archive {
	isFilled := true
	_, err := os.Stat(archiveDir(id))
	if os.IsNotExist(err) {
		isFilled = false
	}
	return &archive{
		id:       id,
		isFilled: isFilled,
	}
}

func (a *archive) isEmpty() bool {
	return !a.isFilled
}

// archive stores the cache record to disk as a set of gob files
func (a *archive) setResult(result *CacheResult) error {
	if a.isFilled {
		return nil
	}

	// create the directory for the history record
	err := os.MkdirAll(archiveDir(a.id), os.ModePerm)
	if err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	// serialize the data
	// files inside the directory ..../call_id/:
	// header.gob - header
	// meta.gob - meta
	// row_0.gob - first row
	// row_n.gob - n-th row

	// header
	file, err := os.Create(headerFile(a.id))
	if err != nil {
		return fmt.Errorf("os.Create: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(result.Header())
	if err != nil {
		return fmt.Errorf("encoder.Encode: %w", err)
	}

	// meta
	file, err = os.Create(metaFile(a.id))
	if err != nil {
		return err
	}
	defer file.Close()

	encoder = gob.NewEncoder(file)
	err = encoder.Encode(*result.Meta())
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
			chunk, err := result.Rows(chunkStart, chunkEnd)
			if err != nil {
				return err
			}
			if len(chunk) == 0 {
				return nil
			}

			file, err := os.Create(rowFile(a.id, i))
			if err != nil {
				return fmt.Errorf("os.Create: %w", err)
			}
			defer file.Close()

			encoder := gob.NewEncoder(file)
			err = encoder.Encode(chunk)
			if err != nil {
				return fmt.Errorf("encoder.Encode: %w", err)
			}

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	a.isFilled = true

	return nil
}

// unarchive loads result from archive in form of an iterator
func (a *archive) getResult() (*archiveRows, error) {
	if !a.isFilled {
		return nil, errors.New("archive does not contain a result")
	}
	return newArchiveRows(a.id)
}

type archiveRows struct {
	id      StatID
	header  models.Header
	meta    *models.Meta
	iter    func() (models.Row, error)
	hasNext func() bool
}

func newArchiveRows(id StatID) (*archiveRows, error) {
	r := &archiveRows{
		id: id,
	}

	err := r.readHeader()
	if err != nil {
		return nil, err
	}
	err = r.readMeta()
	if err != nil {
		return nil, err
	}

	r.readIter()

	return r, nil
}

func (r *archiveRows) readHeader() error {
	// header
	var header models.Header
	file, err := os.Open(headerFile(r.id))
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&header)
	if err != nil {
		return fmt.Errorf("decoder.Decode: %w", err)
	}

	r.header = header

	return nil
}

func (r *archiveRows) readMeta() error {
	// meta
	var meta models.Meta
	file, err := os.Open(metaFile(r.id))
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&meta)
	if err != nil {
		return fmt.Errorf("decoder.Decode: %w", err)
	}

	r.meta = &meta

	return nil
}

func (r *archiveRows) readIter() {
	// open the first file if it exists,
	// loop through its contents and try the next file
	fileExists := func(rowIndex int) bool {
		_, err := os.Stat(rowFile(r.id, rowIndex))
		return err == nil
	}

	// nextFile returns the contents of the next rows file
	index := 0
	nextFile := func() (resultRows []models.Row, isLast bool, err error) {
		file, err := os.Open(rowFile(r.id, index))
		if err != nil {
			return nil, false, fmt.Errorf("os.Open: %w", err)
		}
		defer file.Close()

		var rows []models.Row

		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&rows)
		if err != nil {
			return nil, false, fmt.Errorf("decoder.Decode: %w", err)
		}

		index++
		return rows, !fileExists(index + 1), nil
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
			currentRows, isLastFile, err = nextFile()
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
