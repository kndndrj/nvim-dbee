package conn

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

var ErrInvalidRange = func(from int, to int) error { return fmt.Errorf("invalid selection range: %d ... %d", from, to) }

type cacheRecord struct {
	result  models.Result
	drained bool
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
	records cacheMap
	log     models.Logger
}

func NewCache(pageSize int, logger models.Logger) *cache {
	return &cache{
		records: cacheMap{},
		log:     logger,
	}
}

// Set sets a new record in cache
func (c *cache) Set(ctx context.Context, iter models.IterResult, blockUntil int, id string) error {
	// function to call on fail
	fail := func() {
		iter.Close()
	}

	header, err := iter.Header()
	if err != nil {
		fail()
		return err
	}

	meta, err := iter.Meta()
	if err != nil {
		fail()
		return err
	}

	// create a new result
	result := models.Result{}
	result.Header = header
	result.Meta = meta

	// set context to draining
	_ = contextUpdateCallState(ctx, CallStateCaching)

	c.log.Debug("processing result iterator start: " + id)

	// drain the iterator
	drained := make(chan bool)
	go func() {
		setDrained := func() {
			select {
			case drained <- true:
			default:
			}
		}

		defer func() { setDrained() }()

		i := 0
		for {
			// check if context is valid
			select {
			case <-ctx.Done():
				iter.Close()
				c.log.Warn("operation cancled")
				return
			default:
			}

			// update records in chunks
			if i >= blockUntil {
				c.records.store(id, cacheRecord{
					result: result,
				})
				i = 0

				setDrained()
			}

			row, err := iter.Next()
			if err != nil {
				fail()
				c.log.Error(err.Error())
				return
			}
			if row == nil {
				_ = contextUpdateCallState(ctx, CallStateCached)
				c.log.Debug("successfully exhausted iterator: " + id)
				break
			}

			result.Rows = append(result.Rows, row)
			i++
		}

		// store one last time and set drained to true
		c.records.store(id, cacheRecord{
			drained: true,
			result:  result,
		})
	}()

	// wait until "blockUntil" or all results drained
	<-drained

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
// wipe - deletes the cache after paging
// outputs - where to pipe the results
//
// returns a number of records
func (c *cache) Get(ctx context.Context, id string, from int, to int, outputs ...Output) (int, error) {
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

	var cachedResult models.Result

	timeoutContext, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Wait for drain, available index or timeout
	for {
		rec, ok := c.records.load(id)
		if !ok {
			return 0, fmt.Errorf("record %s appears to be already flushed", id)
		}

		if rec.drained || (to >= 0 && to <= len(rec.result.Rows)) {
			cachedResult = rec.result
			break
		}

		if err := timeoutContext.Err(); err != nil {
			return 0, fmt.Errorf("cache flushing timeout exceeded: %s", err)
		}
		time.Sleep(1 * time.Second)
	}

	// calculate range

	length := len(cachedResult.Rows)
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
	result.Header = cachedResult.Header
	result.Meta = cachedResult.Meta

	result.Rows = cachedResult.Rows[from:to]
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

// Wipe the record with id from cache
func (c *cache) Wipe(id string) {
	c.records.delete(id)
	c.log.Debug("successfully wiped record from cache")
}
