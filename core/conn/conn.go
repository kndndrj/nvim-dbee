package conn

import (
	"log"
)

// types and interfaces
type (
	// Row and Header are attributes of IterResult iterator
	Row    []any
	Header []string
)

type (
	// Schema is a map which represents a database structure
	// it's primarely used for the tree view
	Schema map[string][]string
)

type (
	// IterResult is an iterator which provides rows and headers from the Input
	IterResult interface {
		Header() (Header, error)
		Next() (Row, error)
		Close()
	}

	// Result is the "drained" form of the IterResult iterator used by Output
	Result struct {
		Header Header
		Rows   []Row
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
		List() []string
	}

	// Client is a special kind of input with extra stuff
	Client interface {
		Input
		Close()
		Schema() (Schema, error)
	}
)

type Conn struct {
	driver   Client
	cache    *cache
	pageSize int
	history  History
}

func New(driver Client, pageSize int, history History) *Conn {

	return &Conn{
		pageSize: pageSize,
		driver:   driver,
		history:  history,
		cache:    newCache(pageSize),
	}
}

func (c *Conn) Execute(query string) error {

	log.Print("executing query: \"" + query + "\"")

	rows, err := c.driver.Query(query)
	if err != nil {
		return err
	}

	c.cache.flush(c.history)

	return c.cache.set(rows)
}

func (c *Conn) History(historyId string) error {

	rows, err := c.history.Query(historyId)
	if err != nil {
		return err
	}

	c.cache.flush(c.history)

	return c.cache.set(rows)
}

func (c *Conn) ListHistory() []string {
	return c.history.List()
}

func (c *Conn) Display(page int, outputs ...Output) (int, error) {
	return c.cache.page(page, outputs...)
}

func (c *Conn) Schema() (Schema, error) {
	return c.driver.Schema()
}

func (c *Conn) Close() {
	c.driver.Close()
}
