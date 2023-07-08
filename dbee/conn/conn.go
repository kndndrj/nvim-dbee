package conn

import (
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

type (
	// Input requires implementaions to provide iterator from a
	// given string input, which can be a query or some sort of id
	Input interface {
		Query(string) (models.IterResult, error)
	}

	// Output recieves a result and does whatever it wants with it
	Output interface {
		Write(result models.Result) error
	}

	// History is required to act as an input, output and provide a List method
	History interface {
		Output
		Input
		Layout() ([]models.Layout, error)
	}

	// Client is a special kind of input with extra stuff
	Client interface {
		Input
		Close()
		Layout() ([]models.Layout, error)
	}
)

type Conn struct {
	driver                Client
	cache                 *cache
	currentCachedResultId string
	// how many results to wait for in the main thread? -> see cache.go
	blockUntil int
	history    History
	log        models.Logger
	// is the result fresh (e.g. is it not history?)
	fresh bool
}

func New(driver Client, blockUntil int, history History, logger models.Logger) *Conn {
	return &Conn{
		blockUntil: blockUntil,
		driver:     driver,
		history:    history,
		cache:      NewCache(blockUntil, logger),
		log:        logger,
	}
}

func (c *Conn) Execute(query string) error {
	c.log.Debug("executing query: \"" + query + "\"")

	rows, err := c.driver.Query(query)
	if err != nil {
		return err
	}

	return c.setResultToCache(rows, true)
}

func (c *Conn) History(historyId string) error {
	c.log.Debug("retrieving history with id: \"" + historyId + "\"")

	rows, err := c.history.Query(historyId)
	if err != nil {
		return err
	}

	return c.setResultToCache(rows, false)
}

func (c *Conn) setResultToCache(rows models.IterResult, fresh bool) error {
	// save the old record into history and remove it from cache
	oldID := c.currentCachedResultId
	isFresh := c.fresh
	go func() {
		if isFresh {
			_, err := c.cache.Get(oldID, 0, -1, c.history)
			if err != nil {
				c.log.Debug("failed flushing result to history: " + err.Error())
			}
		}
		c.cache.Wipe(oldID)
	}()

	c.fresh = fresh

	// set new record in cache
	id, err := c.cache.Set(rows, c.blockUntil)
	if err != nil {
		return err
	}
	c.currentCachedResultId = id
	return nil
}

func (c *Conn) ListHistory() ([]models.Layout, error) {
	return c.history.Layout()
}

// GetCurrentResult pipes the selected range of rows to the outputs
// returns length of the result set
func (c *Conn) GetCurrentResult(from int, to int, outputs ...Output) (int, error) {
	return c.cache.Get(c.currentCachedResultId, from, to, outputs...)
}

func (c *Conn) Layout() ([]models.Layout, error) {
	structure, err := c.driver.Layout()
	if err != nil {
		return nil, err
	}
	history, err := c.history.Layout()
	if err != nil {
		return nil, err
	}

	layout := []models.Layout{
		{
			Name:     "structure",
			Schema:   "",
			Database: "",
			Type:     models.LayoutNone,
			Children: structure,
		},
		{
			Name:     "history",
			Schema:   "",
			Database: "",
			Type:     models.LayoutNone,
			Children: history,
		},
	}

	return layout, nil
}

func (c *Conn) Close() {
	if c.fresh {
		c.log.Debug("flushing history on close")
		_, err := c.cache.Get(c.currentCachedResultId, 0, -1, c.history)
		if err != nil {
			c.log.Debug("flushing history on close failed: " + err.Error())
		}
	}

	c.driver.Close()
}
