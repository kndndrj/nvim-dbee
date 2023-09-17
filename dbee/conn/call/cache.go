package call

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var ErrInvalidRange = func(from int, to int) error { return fmt.Errorf("invalid selection range: %d ... %d", from, to) }

type Formatter interface {
	Format(header models.Header, rows []models.Row, meta *models.Meta) ([]byte, error)
}

// CacheResult is the cached form of the Result iterator
type CacheResult struct {
	header models.Header
	meta   *models.Meta
	rows   []models.Row

	isDrained  bool
	isFilled   bool
	writeMutex sync.Mutex
	readMutex  sync.RWMutex
}

func (cr *CacheResult) setIter(iter models.IterResult) error {
	// lock write mutex
	cr.writeMutex.Lock()
	defer cr.writeMutex.Unlock()

	// function to call on fail
	var err error
	defer func() {
		if err != nil {
			iter.Close()
		}
	}()

	cr.isDrained = false
	cr.isFilled = true

	cr.header = iter.Header()
	cr.meta = iter.Meta()
	cr.rows = []models.Row{}

	// drain the iterator
	for iter.HasNext() {
		row, err := iter.Next()
		if err != nil {
			return err
		}

		cr.rows = append(cr.rows, row)
	}

	cr.isDrained = true

	return nil
}

func (cr *CacheResult) Wipe() {
	// lock write and read mutexes
	cr.writeMutex.Lock()
	defer cr.writeMutex.Unlock()
	cr.readMutex.Lock()
	defer cr.readMutex.Unlock()

	*cr = CacheResult{}
	cr.isDrained = false
	cr.isFilled = false
}

func (cr *CacheResult) Format(formatter Formatter, from, to int) ([]byte, error) {
	rows, err := cr.Rows(from, to)
	if err != nil {
		return nil, fmt.Errorf("cr.Rows: %w", err)
	}

	f, err := formatter.Format(cr.header, rows, cr.meta)
	if err != nil {
		return nil, fmt.Errorf("formatter.Format: %w", err)
	}

	return f, nil
}

func (cr *CacheResult) Len() int {
	return len(cr.rows)
}

func (cr *CacheResult) IsEmpty() bool {
	return !cr.isFilled
}

func (cr *CacheResult) Header() models.Header {
	return cr.header
}

func (cr *CacheResult) Meta() *models.Meta {
	return cr.meta
}

func (cr *CacheResult) Rows(from, to int) ([]models.Row, error) {
	// increment the read mutex
	cr.readMutex.RLock()
	defer cr.readMutex.RUnlock()

	// validation
	if (from < 0 && to < 0) || (from >= 0 && to >= 0) {
		if from > to {
			return nil, ErrInvalidRange(from, to)
		}
	}
	// undefined -> error
	if from < 0 && to >= 0 {
		return nil, ErrInvalidRange(from, to)
	}

	// timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Wait for drain, available index or timeout
	for {
		if cr.isDrained || (to >= 0 && to <= len(cr.rows)) {
			break
		}

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("cache flushing timeout exceeded: %s", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// calculate range
	length := len(cr.rows)
	if from < 0 {
		from += length + 1
		if from < 0 {
			from = 0
		}
	}
	if to < 0 {
		to += length + 1
		if to < 0 {
			to = 0
		}
	}

	if from > length {
		from = length
	}
	if to > length {
		to = length
	}

	return cr.rows[from:to], nil
}
