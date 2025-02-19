package testhelpers

import (
	"context"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	tc "github.com/testcontainers/testcontainers-go"
	tcpsql "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type PostgresContainer struct {
	*tcpsql.PostgresContainer
	ConnURL string
	Driver  *core.Connection
}

// NewPostgresContainer creates a new postgres container with
// default adapter and connection. The params.URL is overwritten.
func NewPostgresContainer(ctx context.Context, params *core.ConnectionParams) (*PostgresContainer, error) {
	seedFile, err := GetTestDataFile("postgres_seed.sql")
	if err != nil {
		return nil, err
	}

	ctr, err := tcpsql.Run(
		ctx,
		"postgres:16-alpine",
		tcpsql.BasicWaitStrategies(),
		tc.CustomizeRequest(tc.GenericContainerRequest{
			ProviderType: GetContainerProvider(),
		}),
		tcpsql.WithInitScripts(seedFile.Name()),
		tcpsql.WithDatabase("dev"),
	)
	if err != nil {
		return nil, err
	}
	connURL, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}

	if params.Type == "" {
		params.Type = "postgres"
	}

	if params.URL == "" {
		params.URL = connURL
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &PostgresContainer{
		PostgresContainer: ctr,
		ConnURL:           connURL,
		Driver:            driver,
	}, nil
}

// NewDriver helper function to create a new driver with the connection URL.
func (p *PostgresContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.URL == "" {
		params.URL = p.ConnURL
	}
	if params.Type == "" {
		params.Type = "postgres"
	}

	return adapters.NewConnection(params)
}
