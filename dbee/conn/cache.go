package conn

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
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
// returns id of the newly created records as a response
func (c *cache) Set(iter models.IterResult, blockUntil int) (string, error) {
	// close the iterator on error
	var err error
	defer func() {
		if err != nil {
			iter.Close()
		}
	}()

	header, err := iter.Header()
	if err != nil {
		return "", err
	}

	meta, err := iter.Meta()
	if err != nil {
		return "", err
	}

	// create a new result
	result := models.Result{}
	result.Header = header
	result.Meta = meta

	// create a new id and set it as active
	id := uuid.New().String()
	c.log.Debug("processing result iterator start: " + id)

	// drain the iterator
	done := make(chan bool)
	go func() {
		setDone := func() {
			select {
			case done <- true:
			default:
			}
		}

		defer func() { setDone() }()

		i := 0
		for {
			// update records in chunks
			if i >= blockUntil {
				c.records.store(id, cacheRecord{
					result: result,
				})
				i = 0

				setDone()
			}

			row, err := iter.Next()
			if err != nil {
				c.log.Error(err.Error())
				return
			}
			if row == nil {
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
	<-done

	return id, nil
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
func (c *cache) Get(id string, from int, to int, outputs ...Output) (int, error) {
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

	// timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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

		if err := ctx.Err(); err != nil {
			return 0, fmt.Errorf("cache flushing timeout exceeded: %s", err)
		}
		time.Sleep(1 * time.Second)
	}

	// calculate range

	length := len(cachedResult.Rows)
	if from < 0 {
		from += length
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
		err := out.Write(result)
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
