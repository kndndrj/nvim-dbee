package call

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var (
	ErrInvalidRange       = func(from int, to int) error { return fmt.Errorf("invalid selection range: %d ... %d", from, to) }
	ErrCacheAlreadyFilled = errors.New("cache is already filled")
)

// token is a stealable mutex
type token struct {
	current int
}

func (r *token) Steal() int {
	id := rand.Int()
	r.current = id
	return id
}

func (r *token) Check(id int) bool {
	return r.current == id
}

// cachedResult is the "drained" form of the Result iterator
type cachedResult struct {
	Header models.Header
	Rows   []models.Row
	Meta   models.Meta
}

type CacheState int

const (
	CacheStateEmpty CacheState = iota
	CacheStateFilling
	CacheStateFilled
	CacheStateFailed
)

// cache is a subcomponent of Call, which holds the result in memory
type cache struct {
	result       cachedResult
	state        CacheState
	historyState CacheState
	historyDir   string
	log          models.Logger
	token        *token
}

func NewCache(archivePath string, logger models.Logger) *cache {
	c := &cache{
		state:        CacheStateEmpty,
		log:          logger,
		historyDir:   archivePath,
		historyState: CacheStateEmpty,
		token:        &token{},
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

	// steal the write token
	tokenID := c.token.Steal()

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

			if !c.token.Check(tokenID) {
				fmt.Println("quiting")
				iter.Close()
				return
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
func (c *cache) Get(ctx context.Context, from int, to int) (*cacheRows, error) {
	if !c.HasResultInMemory() && !c.HasResultInArchive() {
		return nil, errors.New("no result")
	}

	// check history
	if !c.HasResultInMemory() && c.HasResultInArchive() {
		err := c.unarchive(ctx)
		if err != nil {
			return nil, err
		}
	}

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

	timeoutContext, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// TODO: this can be made more efficient
	// Wait for drain, available index or timeout
	for {
		if c.state == CacheStateFailed {
			return nil, errors.New("filling cache failed")
		}
		l := len(c.result.Rows)
		if c.state == CacheStateFilled || (c.state == CacheStateFilling && to <= l && to >= 0) {
			break
		}

		// check timeout
		if err := timeoutContext.Err(); err != nil {
			return nil, fmt.Errorf("accessing cache timeout exceeded: %w", err)
		}
		// check context
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("accessing cache cancled: %w", err)
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
	var result cachedResult
	result.Header = c.result.Header
	result.Meta = c.result.Meta

	result.Rows = c.result.Rows[from:to]
	result.Meta.ChunkStart = from
	result.Meta.TotalLength = length

	return newCacheRows(result), nil
}

func (c *cache) Wipe() {
	_ = c.token.Steal()

	c.result = cachedResult{}
	c.state = CacheStateEmpty
}

type cacheRows struct {
	result cachedResult
	index  int
}

func newCacheRows(result cachedResult) *cacheRows {
	return &cacheRows{
		result: result,
	}
}

func (r *cacheRows) Meta() (models.Meta, error) {
	return r.result.Meta, nil
}

func (r *cacheRows) Header() (models.Header, error) {
	return r.result.Header, nil
}

func (r *cacheRows) Next() (models.Row, error) {
	if r.index >= len(r.result.Rows) {
		return nil, nil
	}

	row := r.result.Rows[r.index]
	r.index++

	return row, nil
}

func (r *cacheRows) Close() {}
