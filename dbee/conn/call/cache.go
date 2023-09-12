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
	Meta   *models.Meta
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
	result       *cachedResult
	state        CacheState
	historyState CacheState
	historyDir   string
	log          models.Logger
	token        *token
}

func NewCache(archivePath string, logger models.Logger) *cache {
	c := &cache{
		result:       &cachedResult{},
		state:        CacheStateEmpty,
		historyState: CacheStateEmpty,
		historyDir:   archivePath,
		log:          logger,
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

	// create a new result
	c.result.Header = iter.Header()
	c.result.Meta = iter.Meta()

	// drain the iterator
	go func() {
		for iter.HasNext() {
			row, err := iter.Next()
			if err != nil {
				fail(err)
				return
			}

			if !c.token.Check(tokenID) {
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
		c.state = CacheStateFilled

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
		if c.state == CacheStateFilled || (c.state == CacheStateFilling && to <= len(c.result.Rows) && to >= 0) {
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
	result := &cachedResult{
		Header: c.result.Header,
		Rows:   c.result.Rows[from:to],
		Meta: &models.Meta{
			SchemaType:  c.result.Meta.SchemaType,
			ChunkStart:  from,
			TotalLength: length,
		},
	}

	return newCacheRows(result), nil
}

func (c *cache) Wipe() {
	_ = c.token.Steal()

	c.result = &cachedResult{}
	c.state = CacheStateEmpty
}

type cacheRows struct {
	result *cachedResult
	index  int
}

func newCacheRows(result *cachedResult) *cacheRows {
	return &cacheRows{
		result: result,
	}
}

func (r *cacheRows) Meta() *models.Meta {
	return r.result.Meta
}

func (r *cacheRows) Header() models.Header {
	return r.result.Header
}

func (r *cacheRows) Next() (models.Row, error) {
	if r.index >= len(r.result.Rows) {
		return nil, errors.New("no next row")
	}

	row := r.result.Rows[r.index]
	r.index++

	return row, nil
}

func (r *cacheRows) HasNext() bool {
	return r.index < len(r.result.Rows)
}

func (r *cacheRows) Close() {}
