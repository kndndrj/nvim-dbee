package core

import (
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

func init() {
	// gob doesn't know how to encode/decode time otherwise
	gob.Register(time.Time{})
}

const archiveBasePath = "/tmp/dbee-history/"

// these variables create a file name for a specified type
var (
	archiveDir = func(callID CallID) string {
		return filepath.Join(archiveBasePath, string(callID))
	}

	metaFile = func(callID CallID) string {
		return filepath.Join(archiveDir(callID), "meta.gob")
	}
	headerFile = func(callID CallID) string {
		return filepath.Join(archiveDir(callID), "header.gob")
	}
	rowFile = func(callID CallID, i int) string {
		return filepath.Join(archiveDir(callID), fmt.Sprintf("row_%d.gob", i))
	}
)

type archive struct {
	id       CallID
	isFilled bool
}

func newArchive(id CallID) *archive {
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
func (a *archive) setResult(result *Result) error {
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
	id      CallID
	header  Header
	meta    *Meta
	iter    func() (Row, error)
	hasNext func() bool
}

func newArchiveRows(id CallID) (*archiveRows, error) {
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
	var header Header
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
	var meta Meta
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

// closeOnce closes the channel if it isn't already closed.
func closeOnce[T any](ch chan T) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// readIter creates next and hasNext functions.
// This method is basically the same as builders/NextYield, but is copy-pasted
// because of import cycles.
func (r *archiveRows) readIter() {
	// open the first file if it exists,
	// loop through its contents and try the next file
	fileExists := func(rowIndex int) bool {
		_, err := os.Stat(rowFile(r.id, rowIndex))
		return err == nil
	}

	// openFile returns rows of the file
	openFile := func(i int) ([]Row, error) {
		file, err := os.Open(rowFile(r.id, i))
		if err != nil {
			return nil, fmt.Errorf("os.Open: %w", err)
		}
		defer file.Close()

		var rows []Row

		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&rows)
		if err != nil {
			return nil, fmt.Errorf("decoder.Decode: %w", err)
		}

		return rows, nil
	}

	resultsCh := make(chan []any, 10)
	errorsCh := make(chan error, 1)
	readyCh := make(chan struct{})
	doneCh := make(chan struct{})

	// spawn channel function
	go func() {
		defer func() {
			close(doneCh)
			closeOnce(readyCh)
			close(resultsCh)
			close(errorsCh)
		}()

		file := 0
		for {
			if !fileExists(file) {
				return
			}
			rows, err := openFile(file)
			if err != nil {
				errorsCh <- err
				return
			}

			for _, row := range rows {
				resultsCh <- row
				closeOnce(readyCh)
			}

			file++
		}
	}()

	<-readyCh

	var nextVal atomic.Value
	var nextErr atomic.Value

	r.hasNext = func() bool {
		select {
		case vals, ok := <-resultsCh:
			if !ok {
				return false
			}
			nextVal.Store(vals)
			return true
		case err := <-errorsCh:
			if err != nil {
				nextErr.Store(err)
				return false
			}
		case <-doneCh:
			if len(resultsCh) < 1 {
				return false
			}
		case <-time.After(5 * time.Second):
			nextErr.Store(errors.New("next row timeout"))
			return false
		}

		return r.hasNext()
	}

	r.iter = func() (Row, error) {
		var val Row
		var err error

		nval := nextVal.Load()
		if nval != nil {
			val = nval.([]any)
		}
		nerr := nextErr.Load()
		if nerr != nil {
			err = nerr.(error)
		}
		return val, err
	}
}

func (r *archiveRows) Meta() *Meta {
	return r.meta
}

func (r *archiveRows) Header() Header {
	return r.header
}

func (r *archiveRows) Next() (Row, error) {
	return r.iter()
}

func (r *archiveRows) HasNext() bool {
	return r.hasNext()
}

func (r *archiveRows) Close() {
	// no-op
}
