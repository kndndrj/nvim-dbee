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
	driver   Client
	cache    *cache
	pageSize int
	history  History
	log      models.Logger
	// is the result fresh (e.g. is it not history?)
	fresh bool
}

func New(driver Client, pageSize int, history History, logger models.Logger) *Conn {
	return &Conn{
		pageSize: pageSize,
		driver:   driver,
		history:  history,
		cache:    newCache(pageSize, logger),
		log:      logger,
	}
}

func (c *Conn) Execute(query string) error {
	c.log.Debug("executing query: \"" + query + "\"")

	rows, err := c.driver.Query(query)
	if err != nil {
		return err
	}

	if c.fresh {
		c.cache.flush(true, c.history)
	}

	c.fresh = true

	return c.cache.set(rows)
}

func (c *Conn) History(historyId string) error {
	c.log.Debug("retrieving history with id: \"" + historyId + "\"")

	rows, err := c.history.Query(historyId)
	if err != nil {
		return err
	}

	if c.fresh {
		c.cache.flush(true, c.history)
	}

	c.fresh = false

	return c.cache.set(rows)
}

func (c *Conn) ListHistory() ([]models.Layout, error) {
	return c.history.Layout()
}

// PageCurrent pipes the selected page to the outputs
func (c *Conn) PageCurrent(page int, outputs ...Output) (int, int, error) {
	return c.cache.page(page, outputs...)
}

// SelectRangeCurrent pipes the selected range of rows to the outputs
func (c *Conn) SelectRangeCurrent(from int, to int, outputs ...Output) error {
	return c.cache.span(from, to, outputs...)
}

// WriteCurrent writes the full result to the outputs
func (c *Conn) WriteCurrent(outputs ...Output) error {
	c.cache.flush(false, outputs...)
	return nil
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
		c.cache.flush(true, c.history)
	}

	c.driver.Close()
}
