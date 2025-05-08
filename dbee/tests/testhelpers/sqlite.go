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

type SQLiteContainer struct {
	tc.Container
	ConnURL string
	Driver  *core.Connection
	TempDir string
}

// NewSQLiteContainer creates a new sqlite container with
// default adapter and connection. The params.URL is overwritten.
// It uses a temporary directory (usually the test suite tempDir) to store the db file.
// The tmpDir is then mounted to the container and all the dependencies are installed
// in the container file, while still being able to connect to the db file in the host.
func NewSQLiteContainer(ctx context.Context, params *core.ConnectionParams, tmpDir string) (*SQLiteContainer, error) {
	seedFile, err := GetTestDataFile("sqlite_seed.sql")
	if err != nil {
		return nil, err
	}

	dbName, containerDBPath := "test.db", "/container/db"
	entrypointCmd := []string{
		"apk add sqlite",
		fmt.Sprintf("sqlite3 %s/%s < %s", containerDBPath, dbName, seedFile.Name()),
		"echo 'ready'",
		"tail -f /dev/null", // hack to keep the container running indefinitely
	}

	req := tc.ContainerRequest{
		Image: "alpine:3.21",
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
		WaitingFor: wait.ForLog("ready").WithStartupTimeout(5 * time.Second),
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
		params.Type = "sqlite"
	}

	connURL := filepath.Join(tmpDir, dbName)
	if params.URL == "" {
		params.URL = connURL
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &SQLiteContainer{
		Container: ctr,
		ConnURL:   connURL,
		Driver:    driver,
		TempDir:   tmpDir,
	}, nil
}

// NewDriver helper function to create a new driver with the connection URL.
func (p *SQLiteContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.URL == "" {
		params.URL = p.ConnURL
	}
	if params.Type == "" {
		params.Type = "sqlite"
	}

	return adapters.NewConnection(params)
}
