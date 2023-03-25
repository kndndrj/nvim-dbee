package conn

import (
	"errors"

	"github.com/kndndrj/nvim-dbee/clients"
)

type Result struct {
	Header clients.Header
	Rows   []clients.Row
}

type Span struct {
	From int64
	To   int64
}

type Output interface {
	Write(result Result) error
}

type Conn struct {
	// output Output
	driver clients.Client
	// cache holds the current result
	cache Result
	// currentRows holds the current iterator from the driver
	currentRows clients.Rows
}

func New(driver clients.Client) *Conn {

	return &Conn{
		// output: output,
		driver: driver,
		cache:  Result{},
	}
}

func (c *Conn) pager(rows clients.Rows, span Span) (Result, error) {

	header, err := rows.Header()
	if err != nil {
		return Result{}, err
	}
	if len(header) < 1 {
		return Result{}, errors.New("no headers provided")
	}

	var result Result
	result.Header = header

	for {
		row, err := rows.Next()
		if row == nil {
			break
		}
		if err != nil {
			return Result{}, err
		}

		result.Rows = append(result.Rows, row)
	}

	return result, nil
}

func (c *Conn) Execute(query string, output Output) error {

	rows, err := c.driver.Execute(query)
	if err != nil {
		return err
	}

	result, err := c.pager(rows, Span{From: 0, To: 100})
	if err != nil {
		return err
	}

	c.cache.Header = result.Header
	c.cache.Rows = append(c.cache.Rows, result.Rows...)

	err = output.Write(result)

	return err
}

func (c *Conn) Schema() (clients.Schema, error) {
	return c.driver.Schema()
}

func (c *Conn) Close() {
	c.driver.Close()
}
