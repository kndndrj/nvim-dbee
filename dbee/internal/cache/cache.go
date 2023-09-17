package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/conn/call"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var ErrInvalidRange = func(from int, to int) error { return fmt.Errorf("invalid selection range: %d ... %d", from, to) }

type cacheResult struct {
	header models.Header
	rows   []models.Row
	meta   *models.Meta
}

type cacheRecord struct {
	call     *call.Call
	result   *cacheResult
	drained  bool
	archived bool
}

type cacheMap struct {
	storage sync.Map
}

func (cm *cacheMap) store(key string, value cacheRecord) {
	cm.storage.Store(key, value)
}

func (cm *cacheMap) load(key string) (cacheRecord, bool) {
	val, ok := cm.storage.Load(key)
	if !ok {
		return cacheRecord{}, false
	}

	return val.(cacheRecord), true
}

func (cm *cacheMap) delete(key string) {
	cm.storage.Delete(key)
}

// cache maintains a map of currently active results
// The non active results stay in the list until they are drained
type cache struct {
	records    *cacheMap
	log        models.Logger
	historyDir string
}

func NewCache(logger models.Logger) *cache {
	return &cache{
		records:    new(cacheMap),
		log:        logger,
		historyDir: "/tmp/dbee-history",
	}
}

func (c *cache) SetCall(call *call.Call) error {
	iter, err := call.GetIter()
	if err != nil {
		return fmt.Errorf("call.GetIter: %w", err)
	}
	return c.setResult(call.GetDetails().ID, iter)
}

func (c *cache) setResult(id string, iter models.IterResult) error {
	// create a new result
	result := &cacheResult{
		header: iter.Header(),
		meta:   iter.Meta(),
	}

	// drain the iterator
	go func() {
		const chunkSize = 500
		i := 0
		for iter.HasNext() {
			row, err := iter.Next()
			if err != nil {
				iter.Close()
				c.log.Errorf("draining cache failed: %s", err)
				return
			}

			result.rows = append(result.rows, row)

			// update records in chunks
			if i >= chunkSize {
				c.records.store(id, cacheRecord{
					result: result,
				})
				i = 0
			}

			i++
		}

		// store one last time and set drained to true
		c.records.store(id, cacheRecord{
			drained: true,
			result:  result,
		})

		c.log.Debug("successfully exhausted iterator: " + id)
	}()

	return nil
}

// Get writes the selected line range to outputs
// id - id of the cache record
// from-to - range of rows:
//
//	starts with 0
//	use negative number from the end
//	for example, to pipe all records use: from:0 to:-1
//
// returns a number of records
func (c *cache) GetResult(id string, from int, to int) (models.IterResult, error) {
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

	var result *cacheResult

	// timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Wait for drain, available index or timeout
	for {
		rec, ok := c.records.load(id)
		if !ok {
			return nil, fmt.Errorf("record %q does not exist", id)
		}

		// record not available, so it might be archived
		if rec.result == nil && rec.archived {
			err := c.unarchive()
			if err != nil {
				return nil, err
			}
		}

		if rec.drained || (to >= 0 && to <= len(rec.result.rows)) {
			result = rec.result
			break
		}

		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("cache flushing timeout exceeded: %s", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// calculate range
	length := len(result.rows)
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
	result = &cacheResult{
		header: result.header,
		rows:   result.rows[from:to],
		meta: &models.Meta{
			SchemaType:  result.meta.SchemaType,
			ChunkStart:  from,
			TotalLength: length,
		},
	}

	return newCacheRows(result), nil
}

// Wipe the record with id from cache
func (c *cache) Wipe(id string) {
	c.records.delete(id)
	c.log.Debug("successfully wiped record from cache")
}

// cacheRows implements the IterResult interface
type cacheRows struct {
	result *cacheResult
	index  int
}

func newCacheRows(result *cacheResult) *cacheRows {
	return &cacheRows{
		result: result,
	}
}

func (r *cacheRows) Meta() *models.Meta {
	return r.result.meta
}

func (r *cacheRows) Header() models.Header {
	return r.result.header
}

func (r *cacheRows) Next() (models.Row, error) {
	if r.index >= len(r.result.rows) {
		return nil, errors.New("no next row")
	}

	row := r.result.rows[r.index]
	r.index++

	return row, nil
}

func (r *cacheRows) HasNext() bool {
	return r.index < len(r.result.rows)
}

func (r *cacheRows) Close() {}
