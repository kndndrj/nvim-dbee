package conn

import (
	"errors"
	"fmt"
	"log"

	"github.com/kndndrj/nvim-dbee/clients"
)

type Result struct {
	Header clients.Header
	Rows   []clients.Row
}

type Span struct {
	From int
	To   int
}

type Output interface {
	Write(result Result) error
}

type History interface {
	Output
	clients.Client
}

type Conn struct {
	driver       clients.Client
	currentPager *pager
	pageSize     int
	history      map[string]History
}

func New(driver clients.Client, pageSize int) *Conn {

	return &Conn{
		pageSize: pageSize,
		driver:   driver,
		history:  make(map[string]History),
	}
}

func (c *Conn) Execute(query string) error {

	rows, err := c.driver.Execute(query)
	if err != nil {
		return err
	}

	// create new history record
	h := newHistory()
	// TODO: use some sort of id instead of query
	c.history[query] = h

	// create a new pager and set it as the active one
	pager := newPager(c.pageSize, h)
	c.currentPager = pager

	return pager.set(rows)
}

func (c *Conn) History(query string) error {

	h, ok := c.history[query]
	if !ok {
		return errors.New("no such input in history")
	}

	rows, err := h.Execute("")
	if err != nil {
		return err
	}

	// TODO make history optional in pager
	hi := newHistory()

	// create a new pager and set it as the active one
	pager := newPager(c.pageSize, hi)
	c.currentPager = pager

	return pager.set(rows)
}

func (c *Conn) ListHistory() []string {

	var keys []string
	for k := range c.history {
		keys = append(keys, k)
	}

	return keys
}

func (c *Conn) Display(page int, outputs ...Output) (int, error) {

	result, currentPage, err := c.currentPager.get(page)
	if err != nil {
		return 0, err
	}

	// write result to all outputs
	for _, out := range outputs {
		err = out.Write(result)
		if err != nil {
			return 0, err
		}
	}

	return currentPage, nil
}

func (c *Conn) Schema() (clients.Schema, error) {
	return c.driver.Schema()
}

func (c *Conn) Close() {
	c.driver.Close()
}

// pager is used to "page" through results
// not thread-safe!
type pager struct {
	cache    Result
	history  Output
	pageSize int
}

func newPager(pageSize int, history Output) *pager {
	return &pager{
		cache:    Result{},
		history:  history,
		pageSize: pageSize,
	}
}

func (p *pager) set(iter clients.Rows) error {
	header, err := iter.Header()
	if err != nil {
		return err
	}
	if len(header) < 1 {
		return errors.New("no headers provided")
	}

	// fill the cache
	p.cache = Result{}
	p.cache.Header = header

	// produce the first page
	for i := 0; i < p.pageSize; i++ {
		row, err := iter.Next()
		if row == nil {
			return nil
		}
		if err != nil {
			return err
		}

		p.cache.Rows = append(p.cache.Rows, row)
	}

	// process everything else in a seperate goroutine
	go func() {
		for {
			row, err := iter.Next()
			if err != nil {
				log.Print(err)
				return
			}
			if row == nil {
				log.Print("successfully exhausted iterator")
				log.Print(len(p.cache.Rows))
				break
			}
			p.cache.Rows = append(p.cache.Rows, row)
		}
		err = p.history.Write(p.cache)
		if err != nil {
			log.Print(err)
		}
		log.Print("successfully written history")
	}()

	return nil
}

// page - zero based index of page
// returns current page
func (p *pager) get(page int) (Result, int, error) {
	if p.cache.Header == nil {
		return Result{}, 0, errors.New("no results to page")
	}

	var result Result
	result.Header = p.cache.Header

	if page < 0 {
		page = 0
	}

	start := p.pageSize * page
	end := p.pageSize * (page + 1)

	l := len(p.cache.Rows)
	if start >= l {
		lastPage := l / p.pageSize
		if l%p.pageSize == 0 && lastPage != 0 {
			lastPage -= 1
		}
		start = lastPage * p.pageSize
	}
	if end > l {
		end = l
	}

	fmt.Printf("start: %d", start)
	fmt.Printf("end: %d", end)

	result.Rows = p.cache.Rows[start:end]

	currentPage := start / p.pageSize
	return result, currentPage, nil
}
