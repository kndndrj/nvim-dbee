package testhelpers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// DuckDBContainer represents an in-memory DuckDB instance.
type DuckDBContainer struct {
	tc.Container
	ConnURL string
	Driver  *core.Connection
	TempDir string
}

// NewDuckDBContainer creates a new duckdb container with
// default adapter and connection. The params.URL is overwritten.
// It uses a temporary directory (usually the test suite tempDir) to store the db file.
// The tmpDir is then mounted to the container and all the dependencies are installed
// in the container file, while still being able to connect to the db file in the host.
func NewDuckDBContainer(ctx context.Context, params *core.ConnectionParams, tmpDir string) (*DuckDBContainer, error) {
	seedFile, err := GetTestDataFile("duckdb_seed.sql")
	if err != nil {
		return nil, err
	}

	dbName, containerDBPath := "test_container.db", "/container/db"
	entrypointCmd := []string{
		"apt-get update",
		"apt-get install -y curl",
		"curl https://install.duckdb.org | sh",
		"export PATH='/root/.duckdb/cli/latest':$PATH",
		fmt.Sprintf("duckdb %s/%s < %s", containerDBPath, dbName, seedFile.Name()),
		"echo 'ready'",
		"tail -f /dev/null", // hack to keep the container running indefinitely
	}

	req := tc.ContainerRequest{
		Image: "debian:12.10-slim",
		Files: []tc.ContainerFile{
			{
				Reader:            seedFile,
				ContainerFilePath: seedFile.Name(),
				FileMode:          0o755,
			},
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.Binds = append(hc.Binds, fmt.Sprintf("%s:%s", tmpDir, containerDBPath))
		},
		Cmd:        []string{"sh", "-c", strings.Join(entrypointCmd, " && ")},
		WaitingFor: wait.ForLog("ready").WithStartupTimeout(60 * time.Second),
	}

	ctr, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		ProviderType:     GetContainerProvider(),
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	if params.Type == "" {
		params.Type = "duckdb"
	}
	connURL := filepath.Join(tmpDir, dbName)
	if params.URL == "" {
		params.URL = connURL
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &DuckDBContainer{
		Container: ctr,
		ConnURL:   connURL,
		Driver:    driver,
		TempDir:   tmpDir,
	}, nil
}

// NewDriver helper function to create a new driver with the connection URL.
func (d *DuckDBContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.Type == "" {
		params.Type = "duckdb"
	}
	if params.URL != "" {
		params.URL = d.ConnURL
	}

	return adapters.NewConnection(params)
}
