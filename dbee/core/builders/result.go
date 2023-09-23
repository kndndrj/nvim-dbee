package builders

import (
	"errors"
	"sync"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// Result fills conn.IterResult interface for all sql dbs
type Result struct {
	next     func() (core.Row, error)
	hasNext  func() bool
	close    func()
	callback func()
	meta     *core.Meta
	header   core.Header
	once     sync.Once
}

func (r *Result) SetCustomHeader(header core.Header) {
	r.header = header
}

func (r *Result) SetCallback(callback func()) {
	r.callback = callback
}

func (r *Result) Meta() *core.Meta {
	return r.meta
}

func (r *Result) Header() core.Header {
	return r.header
}

func (r *Result) HasNext() bool {
	return r.hasNext()
}

func (r *Result) Next() (core.Row, error) {
	rows, err := r.next()
	if err != nil || rows == nil {
		r.Close()
		return nil, err
	}
	return rows, nil
}

func (r *Result) Close() {
	r.close()
	if r.callback != nil {
		r.once.Do(r.callback)
	}
	r.hasNext = func() bool {
		return false
	}
}

// ResultBuilder builds the rows
type ResultBuilder struct {
	next    func() (core.Row, error)
	hasNext func() bool
	header  core.Header
	close   func()
	meta    *core.Meta
}

func NewResultBuilder() *ResultBuilder {
	return &ResultBuilder{
		next:    func() (core.Row, error) { return nil, errors.New("no next row") },
		hasNext: func() bool { return false },
		header:  core.Header{},
		close:   func() {},
		meta:    &core.Meta{},
	}
}

func (b *ResultBuilder) WithNextFunc(fn func() (core.Row, error), has func() bool) *ResultBuilder {
	b.next = fn
	b.hasNext = has
	return b
}

func (b *ResultBuilder) WithHeader(header core.Header) *ResultBuilder {
	b.header = header
	return b
}

func (b *ResultBuilder) WithCloseFunc(fn func()) *ResultBuilder {
	b.close = fn
	return b
}

func (b *ResultBuilder) WithMeta(meta *core.Meta) *ResultBuilder {
	b.meta = meta
	return b
}

func (b *ResultBuilder) Build() *Result {
	return &Result{
		next:    b.next,
		hasNext: b.hasNext,
		header:  b.header,
		close:   b.close,
		meta:    b.meta,
		once:    sync.Once{},
	}
}
