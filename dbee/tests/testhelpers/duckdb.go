package testhelpers

import (
	"io"
	"log"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// DuckDBContainer represents an in-memory DuckDB instance.
type DuckDBContainer struct {
	Driver *core.Connection
}

// NewDuckDBContainer creates a new in-memory DuckDB instance.
func NewDuckDBContainer(params *core.ConnectionParams) (*DuckDBContainer, error) {
	seedFile, err := GetTestDataFile("duckdb_seed.sql")
	if err != nil {
		return nil, err
	}
	// Read the file contents into a string
	content, err := io.ReadAll(seedFile)
	if err != nil {
		return nil, err
	}
	seedQuery := string(content)

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

	call := driver.Execute(seedQuery, nil)
	select {
	case <-call.Done():
		err := call.Err()
		if err != nil {
			log.Fatal(err)
		}
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
