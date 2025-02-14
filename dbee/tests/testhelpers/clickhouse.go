package testhelpers

import (
	"context"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

type ClickHouseContainer struct {
	*clickhouse.ClickHouseContainer
	ConnURL string
	Driver  *core.Connection
}

// NewClickHouseContainer creates a new clickhouse container with
// default adapter and connection. The params.URL is overwritten.
func NewClickHouseContainer(ctx context.Context, params *core.ConnectionParams) (*ClickHouseContainer, error) {
	seedFile, err := GetTestDataFile("clickhouse_seed.sql")
	if err != nil {
		return nil, err
	}

	ctr, err := clickhouse.Run(
		ctx,
		"clickhouse/clickhouse-server:25.1-alpine",
		tc.CustomizeRequest(tc.GenericContainerRequest{
			ProviderType: GetContainerProvider(),
		}),
		clickhouse.WithUsername("admin"),
		clickhouse.WithPassword(""),
		clickhouse.WithDatabase("dev"),
		clickhouse.WithInitScripts(seedFile.Name()),
	)
	if err != nil {
		return nil, err
	}

	connURL, err := ctr.ConnectionString(ctx)
	if err != nil {
		return nil, err
	}

	if params.Type == "" {
		params.Type = "clickhouse"
	}

	if params.URL == "" {
		params.URL = connURL
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &ClickHouseContainer{
		ClickHouseContainer: ctr,
		ConnURL:             connURL,
		Driver:              driver,
	}, nil
}

// NewDriver helper function to create a new driver with the connection URL.
func (p *ClickHouseContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.URL == "" {
		params.URL = p.ConnURL
	}
	if params.Type == "" {
		params.Type = "clickhouse"
	}

	return adapters.NewConnection(params)
}
