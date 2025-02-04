package testhelpers

import (
	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// DuckDBContainer represents an in-memory DuckDB instance.
type DuckDBContainer struct {
	Driver *core.Connection
}

// NewDuckDBContainer creates a new in-memory DuckDB instance.
func NewDuckDBContainer(params *core.ConnectionParams) (*DuckDBContainer, error) {
	if params.Type == "" {
		params.Type = "duckdb"
	}
	if params.URL != "" {
		params.URL = ""
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &DuckDBContainer{
		Driver: driver,
	}, nil
}

// NewDriver helper function to create a new driver with the same connection.
func (d *DuckDBContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.Type == "" {
		params.Type = "duckdb"
	}
	if params.URL != "" {
		params.URL = ""
	}

	return adapters.NewConnection(params)
}
