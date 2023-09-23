package builders

import (
	"errors"
	"sync"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

var _ core.ResultStream = (*ResultStream)(nil)

type ResultStream struct {
	next     func() (core.Row, error)
	hasNext  func() bool
	close    func()
	callback func()
	meta     *core.Meta
	header   core.Header
	once     sync.Once
}

func (r *ResultStream) SetCustomHeader(header core.Header) {
	r.header = header
}

func (r *ResultStream) SetCallback(callback func()) {
	r.callback = callback
}

func (r *ResultStream) Meta() *core.Meta {
	return r.meta
}

func (r *ResultStream) Header() core.Header {
	return r.header
}

func (r *ResultStream) HasNext() bool {
	return r.hasNext()
}

func (r *ResultStream) Next() (core.Row, error) {
	rows, err := r.next()
	if err != nil || rows == nil {
		r.Close()
		return nil, err
	}
	return rows, nil
}

func (r *ResultStream) Close() {
	r.close()
	if r.callback != nil {
		r.once.Do(r.callback)
	}
	r.hasNext = func() bool {
		return false
	}
}

// ResultStreamBuilder builds the rows
type ResultStreamBuilder struct {
	next    func() (core.Row, error)
	hasNext func() bool
	header  core.Header
	close   func()
	meta    *core.Meta
}

func NewResultStreamBuilder() *ResultStreamBuilder {
	return &ResultStreamBuilder{
		next:    func() (core.Row, error) { return nil, errors.New("no next row") },
		hasNext: func() bool { return false },
		header:  core.Header{},
		close:   func() {},
		meta:    &core.Meta{},
	}
}

func (b *ResultStreamBuilder) WithNextFunc(fn func() (core.Row, error), has func() bool) *ResultStreamBuilder {
	b.next = fn
	b.hasNext = has
	return b
}

func (b *ResultStreamBuilder) WithHeader(header core.Header) *ResultStreamBuilder {
	b.header = header
	return b
}

func (b *ResultStreamBuilder) WithCloseFunc(fn func()) *ResultStreamBuilder {
	b.close = fn
	return b
}

func (b *ResultStreamBuilder) WithMeta(meta *core.Meta) *ResultStreamBuilder {
	b.meta = meta
	return b
}

func (b *ResultStreamBuilder) Build() *ResultStream {
	return &ResultStream{
		next:    b.next,
		hasNext: b.hasNext,
		header:  b.header,
		close:   b.close,
		meta:    b.meta,
		once:    sync.Once{},
	}
}
