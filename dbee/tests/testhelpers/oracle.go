package testhelpers

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type OracleContainer struct {
	tc.Container
	ConnURL string
	Driver  *core.Connection
}

// NewOracleContainer creates a new oracle container with
// default adapter and connection. The params.URL is overwritten.
func NewOracleContainer(ctx context.Context, params *core.ConnectionParams) (*OracleContainer, error) {
	const (
		password      = "password"
		appUser       = "tester"
		port          = "1521/tcp"
		memoryLimitGB = 3 * 1024 * 1024 * 1024
	)

	seedFile, err := GetTestDataFile("oracle_seed.sql")
	if err != nil {
		return nil, err
	}

	req := tc.ContainerRequest{
		Image:        "gvenzl/oracle-free:23.6-slim-faststart",
		ExposedPorts: []string{port},
		Env: map[string]string{
			"ORACLE_PASSWORD":   password,
			"APP_USER":          appUser,
			"APP_USER_PASSWORD": password,
		},
		WaitingFor: wait.ForLog("DATABASE IS READY TO USE!").WithStartupTimeout(5 * time.Minute),
		Resources:  container.Resources{Memory: memoryLimitGB},
		Files: []tc.ContainerFile{
			{
				Reader:            seedFile,
				ContainerFilePath: "/docker-entrypoint-initdb.d/" + seedFile.Name(),
				FileMode:          0o755,
			},
		},
	}

	ctr, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		ProviderType:     GetContainerProvider(),
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		return nil, err
	}

	mPort, err := ctr.MappedPort(ctx, port)
	if err != nil {
		return nil, err
	}

	connURL := fmt.Sprintf("oracle://%s:%s@%s:%d/FREEPDB1", appUser, password, host, mPort.Int())
	if params.Type == "" {
		params.Type = "oracle"
	}

	if params.URL == "" {
		params.URL = connURL
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &OracleContainer{ConnURL: connURL, Driver: driver}, nil
}

// NewDriver helper function to create a new driver with the connection URL.
func (p *OracleContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.URL == "" {
		params.URL = p.ConnURL
	}
	if params.Type == "" {
		params.Type = "oracle"
	}

	return adapters.NewConnection(params)
}
