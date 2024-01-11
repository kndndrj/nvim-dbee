package mock

import (
	"context"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

var _ core.Driver = (*driver)(nil)

type driver struct {
	data   []core.Row
	config *adapterConfig
}

func (d *driver) Query(ctx context.Context, query string) (core.ResultStream, error) {
	eff, ok := d.config.querySideEffects[query]
	if ok {
		err := eff(ctx)
		if err != nil {
			return nil, fmt.Errorf("side effect error: %w", err)
		}
	}

	return NewResultStream(d.data, d.config.resultStreamOptions...), nil
}

func (d *driver) Structure() ([]*core.Structure, error) {
	var structure []*core.Structure

	for table := range d.config.tableColumns {
		structure = append(structure, &core.Structure{
			Name: table,
			Type: core.StructureTypeTable,
		})
	}

	return structure, nil
}

func (d *driver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	columns, ok := d.config.tableColumns[opts.Table]
	if !ok {
		return nil, fmt.Errorf("unknown table: %s", opts.Table)
	}

	return columns, nil
}

func (d *driver) Close() {}

var _ core.Adapter = (*Adapter)(nil)

type Adapter struct {
	data   []core.Row
	config *adapterConfig
}

func NewAdapter(data []core.Row, opts ...AdapterOption) *Adapter {
	config := &adapterConfig{
		querySideEffects: make(map[string]func(context.Context) error),
		tableHelpers:     make(map[string]string),
		tableColumns:     make(map[string][]*core.Column),

		resultStreamOptions: []ResultStreamOption{},
	}
	for _, opt := range opts {
		opt(config)
	}

	return &Adapter{
		data:   data,
		config: config,
	}
}

func (a *Adapter) Connect(_ string) (core.Driver, error) {
	return &driver{
		data:   a.data,
		config: a.config,
	}, nil
}

func (a *Adapter) GetHelpers(opts *core.TableOptions) map[string]string {
	return a.config.tableHelpers
}
