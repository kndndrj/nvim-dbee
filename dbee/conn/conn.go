package conn

import (
	"encoding/json"
	"time"
)

// types and interfaces
type (
	// Row and Header are attributes of IterResult iterator
	Row    []any
	Header []string
)

type (
	LayoutType int
	// Layout is a dict which represents a database structure
	// it's primarely used for the tree view
	Layout struct {
		Name     string     `json:"name"`
		Schema   string     `json:"schema"`
		Database string     `json:"database"`
		Type     LayoutType `json:"type"`
		Children []Layout   `json:"children"`
	}
)

const (
	LayoutNone LayoutType = iota
	LayoutTable
	LayoutHistory
)

func (s LayoutType) String() string {
	switch s {
	case LayoutNone:
		return ""
	case LayoutTable:
		return "table"
	case LayoutHistory:
		return "history"
	default:
		return ""
	}
}

func (s *Layout) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name     string   `json:"name"`
		Schema   string   `json:"schema"`
		Database string   `json:"database"`
		Type     string   `json:"type"`
		Children []Layout `json:"children"`
	}{
		Name:     s.Name,
		Schema:   s.Schema,
		Database: s.Database,
		Type:     s.Type.String(),
		Children: s.Children,
	})
}

type (
	// Meta holds metadata
	Meta struct {
		Query     string
		Timestamp time.Time
	}

	// IterResult is an iterator which provides rows and headers from the Input
	IterResult interface {
		Meta() (Meta, error)
		Header() (Header, error)
		Next() (Row, error)
		Close()
	}

	// Result is the "drained" form of the IterResult iterator used by Output
	Result struct {
		Header Header
		Rows   []Row
		Meta   Meta
	}
)

type (
	// Input requires implementaions to provide iterator from a
	// given string input, which can be a query or some sort of id
	Input interface {
		Query(string) (IterResult, error)
	}

	// Output recieves a result and does whatever it wants with it
	Output interface {
		Write(result Result) error
	}

	// History is required to act as an input, output and provide a List method
	History interface {
		Output
		Input
		Layout() ([]Layout, error)
	}

	// Client is a special kind of input with extra stuff
	Client interface {
		Input
		Close()
		Layout() ([]Layout, error)
	}
)

type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

type Conn struct {
	driver   Client
	cache    *cache
	pageSize int
	history  History
	log      Logger
	// is the result fresh (e.g. is it not history?)
	fresh bool
}

func New(driver Client, pageSize int, history History, logger Logger) *Conn {

	return &Conn{
		pageSize: pageSize,
		driver:   driver,
		history:  history,
		cache:    newCache(pageSize, logger),
		log:      logger,
	}
}

func (c *Conn) Execute(query string) error {

	c.log.Info("executing query: \"" + query + "\"")

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

	c.log.Info("retrieving history with id: \"" + historyId + "\"")

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

func (c *Conn) ListHistory() ([]Layout, error) {
	return c.history.Layout()
}

func (c *Conn) PageCurrent(page int, outputs ...Output) (int, int, error) {
	return c.cache.page(page, outputs...)
}

func (c *Conn) WriteCurrent(outputs ...Output) error {
	c.cache.flush(false, outputs...)
	return nil
}

func (c *Conn) Layout() ([]Layout, error) {

	structure, err := c.driver.Layout()
	if err != nil {
		return nil, err
	}
	history, err := c.history.Layout()
	if err != nil {
		return nil, err
	}

	layout := []Layout{
		{
			Name:     "structure",
			Schema:   "",
			Database: "",
			Type:     LayoutNone,
			Children: structure,
		},
		{
			Name:     "history",
			Schema:   "",
			Database: "",
			Type:     LayoutNone,
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
