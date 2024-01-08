package builders

import (
	"errors"
	"sync"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

var _ core.ResultStream = (*ResultStream)(nil)

type ResultStream struct {
	next    func() (core.Row, error)
	hasNext func() bool
	closes  []func()
	meta    *core.Meta
	header  core.Header
	once    sync.Once
}

func (r *ResultStream) AddCallback(fn func()) {
	r.closes = append(r.closes, fn)
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
	r.once.Do(func() {
		for _, fn := range r.closes {
			if fn != nil {
				fn()
			}
		}
	})

	r.hasNext = func() bool {
		return false
	}
}

// ResultStreamBuilder builds the rows
type ResultStreamBuilder struct {
	next    func() (core.Row, error)
	hasNext func() bool
	header  core.Header
	closes  []func()
	meta    *core.Meta
}

func NewResultStreamBuilder() *ResultStreamBuilder {
	return &ResultStreamBuilder{
		next:    func() (core.Row, error) { return nil, errors.New("no next row") },
		hasNext: func() bool { return false },
		header:  core.Header{},
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
	b.closes = append(b.closes, fn)
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
		closes:  b.closes,
		meta:    b.meta,
		once:    sync.Once{},
	}
}
