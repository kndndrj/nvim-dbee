package testhelpers

import (
	"context"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	tc "github.com/testcontainers/testcontainers-go"
	tcmssql "github.com/testcontainers/testcontainers-go/modules/mssql"
)

type MSSQLServerContainer struct {
	*tcmssql.MSSQLServerContainer
	ConnURL string
	Driver  *core.Connection
}

// NewSQLServerContainer creates a new MS SQL Server container with
// default adapter and connection. The params.URL is overwritten.
func NewSQLServerContainer(ctx context.Context, params *core.ConnectionParams) (*MSSQLServerContainer, error) {
	const password = "H3ll0@W0rld"
	seedFile, err := GetTestDataFile("sqlserver_seed.sql")
	if err != nil {
		return nil, err
	}

	ctr, err := tcmssql.Run(
		ctx,
		"mcr.microsoft.com/mssql/server:2022-CU17-ubuntu-22.04",
		tcmssql.WithAcceptEULA(), // ok for testing purposes
		tcmssql.WithPassword(password),
		tc.CustomizeRequest(tc.GenericContainerRequest{
			ContainerRequest: tc.ContainerRequest{
				Files: []tc.ContainerFile{
					{
						Reader:            seedFile,
						ContainerFilePath: seedFile.Name(),
						FileMode:          0o644,
					},
				},
			},
			ProviderType: GetContainerProvider(),
		}),
		tc.WithAfterReadyCommand(
			tc.NewRawCommand([]string{
				"/opt/mssql-tools18/bin/sqlcmd",
				"-S", "localhost",
				"-U", "sa",
				"-P", password,
				"-No",
				"-i", seedFile.Name(),
			}),
		),
	)
	if err != nil {
		return nil, err
	}

	connURL, err := ctr.ConnectionString(ctx, "encrypt=false", "TrustServerCertificate=true")
	if err != nil {
		return nil, err
	}

	if params.Type == "" {
		params.Type = "mssql"
	}

	if params.URL == "" {
		params.URL = connURL
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &MSSQLServerContainer{
		MSSQLServerContainer: ctr,
		ConnURL:              connURL,
		Driver:               driver,
	}, nil
}

// NewDriver helper function to create a new driver with the connection URL.
func (p *MSSQLServerContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.URL == "" {
		params.URL = p.ConnURL
	}
	if params.Type == "" {
		params.Type = "mssql"
	}

	return adapters.NewConnection(params)
}
