package mock

import (
	"context"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

type adapterConfig struct {
	querySideEffects map[string]func(context.Context) error
	tableHelpers     map[string]string
	tableColumns     map[string][]*core.Column

	resultStreamOptions []ResultStreamOption
}

type AdapterOption func(*adapterConfig)

func AdapterWithQuerySideEffect(query string, sideEffect func(context.Context) error) AdapterOption {
	return func(c *adapterConfig) {
		_, ok := c.querySideEffects[query]
		if ok {
			panic("side effect already registered for query: " + query)
		}

		c.querySideEffects[query] = sideEffect
	}
}

func AdapterWithTableHelper(name string, query string) AdapterOption {
	return func(c *adapterConfig) {
		_, ok := c.tableHelpers[name]
		if ok {
			panic("query already registered for table helper: " + name)
		}

		c.tableHelpers[name] = query
	}
}

func AdapterWithTableDefinition(table string, columns []*core.Column) AdapterOption {
	return func(c *adapterConfig) {
		_, ok := c.tableColumns[table]
		if ok {
			panic("columns already registered for table: " + table)
		}

		c.tableColumns[table] = columns
	}
}

func AdapterWithResultStreamOpts(opts ...ResultStreamOption) AdapterOption {
	return func(c *adapterConfig) {
		c.resultStreamOptions = append(c.resultStreamOptions, opts...)
	}
}
