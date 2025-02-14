package testhelpers

import (
	"context"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	tc "github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
)

type MySQLContainer struct {
	*tcmysql.MySQLContainer
	ConnURL string
	Driver  *core.Connection
}

// NewMySQLContainer creates a new MySQL container with
// default adapter and connection. The params.URL is overwritten.
func NewMySQLContainer(ctx context.Context, params *core.ConnectionParams) (*MySQLContainer, error) {
	seedFile, err := GetTestDataFile("mysql_seed.sql")
	if err != nil {
		return nil, err
	}

	ctr, err := tcmysql.Run(
		ctx,
		"mysql:9.2.0",
		tc.CustomizeRequest(tc.GenericContainerRequest{
			ProviderType: GetContainerProvider(),
		}),
		tcmysql.WithDatabase("dev"),
		tcmysql.WithPassword("password"),
		tcmysql.WithUsername("root"),
		tcmysql.WithScripts(seedFile.Name()),
	)
	if err != nil {
		return nil, err
	}

	connURL, err := ctr.ConnectionString(ctx, "tls=skip-verify")
	if err != nil {
		return nil, err
	}

	if params.Type == "" {
		params.Type = "mysql"
	}

	if params.URL == "" {
		params.URL = connURL
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &MySQLContainer{
		MySQLContainer: ctr,
		ConnURL:        connURL,
		Driver:         driver,
	}, nil
}

// NewDriver helper function to create a new driver with the connection URL.
func (p *MySQLContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.URL == "" {
		params.URL = p.ConnURL
	}
	if params.Type == "" {
		params.Type = "mysql"
	}

	return adapters.NewConnection(params)
}
