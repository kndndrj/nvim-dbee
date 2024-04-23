package mock

import (
	"errors"
	"fmt"
	"time"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

func newNext(rows []core.Row) (func() (core.Row, error), func() bool) {
	index := 0

	hasNext := func() bool {
		return index < len(rows)
	}

	// iterator functions
	next := func() (core.Row, error) {
		if !hasNext() {
			return nil, errors.New("no next row")
		}

		row := rows[index]
		index++
		return row, nil
	}

	return next, hasNext
}

type ResultStream struct {
	next    func() (core.Row, error)
	hasNext func() bool
	config  *resultStreamConfig
}

func makeDefaultHeader(rows []core.Row) core.Header {
	var header core.Header
	if len(rows) > 0 {
		for i := range rows[0] {
			header = append(header, fmt.Sprintf("header_%d", i))
		}
	}
	return header
}

// NewResultStream returns a mocked result stream with provided rows.
// It creates a header that matches the number of columns in the first row
// in form of: <header_0>, <header_1>, etc.
func NewResultStream(rows []core.Row, opts ...ResultStreamOption) *ResultStream {
	config := &resultStreamConfig{
		nextSleep: 0,
		meta:      &core.Meta{},
		header:    makeDefaultHeader(rows),
	}
	for _, opt := range opts {
		opt(config)
	}

	next, hasNext := newNext(rows)

	return &ResultStream{
		next:    next,
		hasNext: hasNext,
		config:  config,
	}
}

func (rs *ResultStream) Meta() *core.Meta {
	return rs.config.meta
}

func (rs *ResultStream) Header() core.Header {
	return rs.config.header
}

func (rs *ResultStream) Next() (core.Row, error) {
	time.Sleep(rs.config.nextSleep)
	return rs.next()
}

func (rs *ResultStream) HasNext() bool {
	return rs.hasNext()
}

func (rs *ResultStream) Close() {}

// NewRows returns a slice of rows in form of:
//
//	{ <index>(int), "row_<index>"(string) }
//
// where the first index is "from" and the last one is one less than "to".
func NewRows(from, to int) []core.Row {
	var rows []core.Row

	for i := from; i < to; i++ {
		rows = append(rows, core.Row{i, fmt.Sprintf("row_%d", i)})
	}
	return rows
}
