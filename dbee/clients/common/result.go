package common

import (
	"errors"
	"sync"

	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// Result fills conn.IterResult interface for all sql dbs
type Result struct {
	next     func() (models.Row, error)
	hasNext  func() bool
	close    func()
	callback func()
	meta     *models.Meta
	header   models.Header
	once     sync.Once
}

func (r *Result) SetCustomHeader(header models.Header) {
	r.header = header
}

func (r *Result) SetCallback(callback func()) {
	r.callback = callback
}

func (r *Result) Meta() *models.Meta {
	return r.meta
}

func (r *Result) Header() models.Header {
	return r.header
}

func (r *Result) HasNext() bool {
	return r.hasNext()
}

func (r *Result) Next() (models.Row, error) {
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
	next    func() (models.Row, error)
	hasNext func() bool
	header  models.Header
	close   func()
	meta    *models.Meta
}

func NewResultBuilder() *ResultBuilder {
	return &ResultBuilder{
		next:    func() (models.Row, error) { return nil, errors.New("no next row") },
		hasNext: func() bool { return false },
		header:  models.Header{},
		close:   func() {},
		meta:    &models.Meta{},
	}
}

func (b *ResultBuilder) WithNextFunc(fn func() (models.Row, error), has func() bool) *ResultBuilder {
	b.next = fn
	b.hasNext = has
	return b
}

func (b *ResultBuilder) WithHeader(header models.Header) *ResultBuilder {
	b.header = header
	return b
}

func (b *ResultBuilder) WithCloseFunc(fn func()) *ResultBuilder {
	b.close = fn
	return b
}

func (b *ResultBuilder) WithMeta(meta *models.Meta) *ResultBuilder {
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
