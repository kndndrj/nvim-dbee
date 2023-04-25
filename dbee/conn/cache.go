package conn

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

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
// only one result is the active one (the latest one).
// The non active results stay in the list until they are drained
type cache struct {
	active   string
	records  cacheMap
	pageSize int
	log      models.Logger
}

func newCache(pageSize int, logger models.Logger) *cache {
	return &cache{
		pageSize: pageSize,
		records:  cacheMap{},
		log:      logger,
	}
}

func (c *cache) set(iter models.IterResult) error {

	// close the iterator on error
	var err error
	defer func() {
		if err != nil {
			iter.Close()
		}
	}()

	header, err := iter.Header()
	if err != nil {
		return err
	}
	if len(header) < 1 {
		return errors.New("no headers provided")
	}

	meta, err := iter.Meta()
	if err != nil {
		return err
	}

	// create a new result
	result := models.Result{}
	result.Header = header
	result.Meta = meta

	// produce the first page
	drained := false
	for i := 0; i < c.pageSize; i++ {
		row, err := iter.Next()
		if err != nil {
			return err
		}
		if row == nil {
			drained = true
			c.log.Debug("successfully exhausted iterator")
			break
		}

		result.Rows = append(result.Rows, row)
	}

	// create a new record and set it's id as active
	id := uuid.New().String()
	c.records.store(id, cacheRecord{
		result:  result,
		drained: drained,
	})
	c.active = id

	// process everything else in a seperate goroutine
	if !drained {
		go func() {
			i := 0
			for {
				// update records in chunks
				if i >= c.pageSize {
					c.records.store(id, cacheRecord{
						result: result,
					})
					i = 0
				}
				row, err := iter.Next()
				if err != nil {
					c.log.Error(err.Error())
					return
				}
				if row == nil {
					c.log.Debug("successfully exhausted iterator")
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
	}

	return nil
}

// zero based index of page
// returns current page and total number of pages
// writes the requested page to outputs
func (c *cache) page(page int, outputs ...Output) (int, int, error) {
	id := c.active

	cr, _ := c.records.load(id)
	cachedResult := cr.result

	if cachedResult.Header == nil {
		return 0, 0, errors.New("no results to page")
	}

	var result models.Result
	result.Header = cachedResult.Header
	result.Meta = cachedResult.Meta

	if page < 0 {
		page = 0
	}

	start := c.pageSize * page
	end := c.pageSize * (page + 1)

	l := len(cachedResult.Rows)
	lastPage := l / c.pageSize
	if l%c.pageSize == 0 && lastPage != 0 {
		lastPage -= 1
	}

	if start >= l {
		start = lastPage * c.pageSize
	}
	if end > l {
		end = l
	}

	result.Rows = cachedResult.Rows[start:end]

	// write the page to outputs
	for _, out := range outputs {
		err := out.Write(result)
		if err != nil {
			return 0, 0, err
		}
	}

	currentPage := start / c.pageSize
	return currentPage, lastPage, nil
}

// flush writes the whole current cache to outputs
// wipe controls wheather to wipe the record from cache
func (c *cache) flush(wipe bool, outputs ...Output) {
	id := c.active

	// wait until the currently active record is drained,
	// write it to outputs and remove it from records
	go func() {

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		// Wait for flag to be set or timeout to exceed
		for {

			rec, ok := c.records.load(id)
			if !ok {
				c.log.Error("record " + id + " appears to be already flushed")
				return
			}
			if rec.drained {
				break
			}
			if ctx.Err() != nil {
				c.log.Error("cache flushing timeout exceeded")
				return
			}
			time.Sleep(1 * time.Second)
		}

		// write to outputs
		for _, out := range outputs {
			rec, _ := c.records.load(id)
			err := out.Write(rec.result)
			if err != nil {
				c.log.Error(err.Error())
			}
		}

		if wipe {
			// delete the record
			c.records.delete(id)
			c.log.Debug("successfully wiped record from cache")
		}
		c.log.Debug("successfully flushed cache")
	}()
}
