package call

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var (
	ErrInvalidRange       = func(from int, to int) error { return fmt.Errorf("invalid selection range: %d ... %d", from, to) }
	ErrCacheAlreadyFilled = errors.New("cache is already filled")
)

type CacheState int

const (
	CacheStateEmpty CacheState = iota
	CacheStateFilling
	CacheStateFilled
	CacheStateFailed
)

// cache is a subcomponent of Call, which holds the result in memory
type cache struct {
	result       models.Result
	state        CacheState
	historyState CacheState
	historyDir   string
	log          models.Logger
}

func NewCache(archivePath string, logger models.Logger) *cache {
	c := &cache{
		state:        CacheStateEmpty,
		log:          logger,
		historyDir:   archivePath,
		historyState: CacheStateEmpty,
	}

	_, err := os.Stat(c.historyDir)
	if err == nil {
		c.historyState = CacheStateFilled
	}

	return c
}

func (c *cache) HasResultInMemory() bool {
	return c.state == CacheStateFilled || c.state == CacheStateFilling
}

func (c *cache) HasResultInArchive() bool {
	return c.historyState == CacheStateFilled
}

// Set sets a record to empty cache
func (c *cache) Set(ctx context.Context, iter models.IterResult) error {
	if c.HasResultInMemory() {
		return ErrCacheAlreadyFilled
	}
	c.state = CacheStateFilling

	// function to call on fail
	fail := func(e error) {
		iter.Close()
		c.state = CacheStateFailed
		c.log.Errorf("draining cache failed: %s", e)
	}

	header, err := iter.Header()
	if err != nil {
		fail(err)
		return err
	}

	meta, err := iter.Meta()
	if err != nil {
		fail(err)
		return err
	}

	// create a new result
	c.result.Header = header
	c.result.Meta = meta

	// drain the iterator
	go func() {
		for {
			row, err := iter.Next()
			if err != nil {
				fail(err)
				return
			}
			if row == nil {
				c.state = CacheStateFilled
				break
			}

			c.result.Rows = append(c.result.Rows, row)

			// check if context is still valid
			if err := ctx.Err(); err != nil {
				fail(err)
				return
			}
		}

		// write to history
		_ = c.archive(c.result)
	}()

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
	if !c.HasResultInMemory() && !c.HasResultInArchive() {
		return 0, errors.New("no result")
	}

	// check history
	if !c.HasResultInMemory() && c.HasResultInArchive() {
		err := c.unarchive(ctx)
		if err != nil {
			return 0, err
		}
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
		if c.state == CacheStateFailed {
			return 0, errors.New("filling cache failed")
		}
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

		// arbirtary delay
		time.Sleep(50 * time.Millisecond)
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
	c.state = CacheStateEmpty
}
