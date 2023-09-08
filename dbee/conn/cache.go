package conn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var (
	ErrInvalidRange  = func(from int, to int) error { return fmt.Errorf("invalid selection range: %d ... %d", from, to) }
	ErrAlreadyFilled = errors.New("cache is already filled")
)

type CacheState int

const (
	CacheStateAvailable CacheState = iota
	CacheStateFilling
	CacheStateFilled
	CacheStateFailed
)

// cache is a subcomponent of Call, which holds the result in memory
type cache struct {
	result models.Result
	state  CacheState
}

func NewCache() *cache {
	return &cache{}
}

func (c *cache) HasResult() bool {
	if c.state == CacheStateFilled || c.state == CacheStateFilling {
		return true
	}
	return false
}

// Set sets a record to empty cache
func (c *cache) Set(ctx context.Context, iter models.IterResult) error {
	if c.HasResult() {
		return ErrAlreadyFilled
	}
	c.state = CacheStateFilling

	// function to call on fail
	var err error
	defer func() {
		iter.Close()
		if err != nil {
			c.state = CacheStateFailed
		}
	}()

	header, err := iter.Header()
	if err != nil {
		return err
	}

	meta, err := iter.Meta()
	if err != nil {
		return err
	}

	// create a new result
	c.result.Header = header
	c.result.Meta = meta

	// drain the iterator
	drain := func() error {
		for {
			row, err := iter.Next()
			if err != nil {
				return err
			}
			if row == nil {
				return nil
			}

			c.result.Rows = append(c.result.Rows, row)

			// check if context is still valid
			if err := ctx.Err(); err != nil {
				return err
			}
		}
	}

	err = drain()
	if err != nil {
		return err
	}

	c.state = CacheStateFilled

	return nil
}

// Get writes the selected line range to outputs
// from-to - range of rows:
//
//	starts with 0
//	use negative number from the end
//	for example, to pipe all records use: from:0 to:-1
//
// outputs - where to pipe the results
//
// returns a number of records
func (c *cache) Get(ctx context.Context, from int, to int, outputs ...Output) (int, error) {
	if !c.HasResult() {
		return 0, errors.New("no result")
	}
	// validation
	if (from < 0 && to < 0) || (from >= 0 && to >= 0) {
		if from > to {
			return 0, ErrInvalidRange(from, to)
		}
	}
	// undefined -> error
	if from < 0 && to >= 0 {
		return 0, ErrInvalidRange(from, to)
	}

	timeoutContext, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Wait for drain, available index or timeout
	for {
		if c.state == CacheStateFilled || (c.state == CacheStateFilling && to <= len(c.result.Rows)) {
			break
		}

		// check timeout
		if err := timeoutContext.Err(); err != nil {
			return 0, fmt.Errorf("accessing cache timeout exceeded: %w", err)
		}
		// check context
		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("accessing cache cancled: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	// calculate range

	length := len(c.result.Rows)
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

	// create a new page
	var result models.Result
	result.Header = c.result.Header
	result.Meta = c.result.Meta

	result.Rows = c.result.Rows[from:to]
	result.Meta.ChunkStart = from

	// write the page to outputs
	for _, out := range outputs {
		err := out.Write(ctx, result)
		if err != nil {
			return 0, err
		}
	}

	return length, nil
}

func (c *cache) Wipe() {
	c.result = models.Result{}
	c.state = CacheStateAvailable
}
