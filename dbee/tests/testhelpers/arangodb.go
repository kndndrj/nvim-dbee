package testhelpers

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// ArangoDBContainer is a test container for ArangoDB.
type ArangoDBContainer struct {
	tc.Container
	ConnURL string
	Driver  *core.Connection
}

type ArangoDBContainerParams struct {
	Passwordless bool
	DatabaseName string
}

// NewArangoDBContainer creates a new ArangoDB container with
// default adapter and connection. The params.URL is overwritten.
func NewArangoDBContainer(ctx context.Context, params *core.ConnectionParams, containerParams *ArangoDBContainerParams) (*ArangoDBContainer, error) {
	seedFile, err := GetTestDataFile("arangodb_seed.json")
	if err != nil {
		return nil, err
	}

	passwordless := false
	if containerParams != nil {
		passwordless = containerParams.Passwordless
	}

	env := make(map[string]string, 0)
	if passwordless {
		env["ARANGO_ROOT_PASSWORD"] = "rootpassword"
	} else {
		env["ARANGO_NO_AUTH"] = "1"
	}

	log.Printf("%s", seedFile.Name())
	req := tc.ContainerRequest{
		Image:        "arangodb:3.12",
		ExposedPorts: []string{"8529:8529/tcp"},
		WaitingFor:   wait.ForLog("ArangoDB (version 3.12.4 [linux]) is ready for business. Have fun!").WithStartupTimeout(1 * time.Minute),
		Env:          env,
		Files: []tc.ContainerFile{
			{
				HostFilePath:      "../testdata/arangodb_seed.json",
				ContainerFilePath: "/docker-entrypoint-initdb.d/arangodb_seed.json",
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

	args := []string{
		"arangoimport",
		"--file", "/docker-entrypoint-initdb.d/arangodb_seed.json",
		"--type", "json",
		"--collection", "testcollection",
		"--create-collection",
	}
	if !passwordless {
		args = append(args, "--server.password", "rootpassword")
	}

	if containerParams != nil {
		if containerParams.DatabaseName != "" && containerParams.DatabaseName != "_system" {
			args = append(args, "--server.database", containerParams.DatabaseName)
			args = append(args, "--create-database")
		}
	}

	exitCode, output, err := ctr.Exec(ctx, args)
	if err != nil {
		return nil, err
	}
	if exitCode != 0 {
		var d string
		if b, err := io.ReadAll(output); err == nil {
			d = string(b)
		}
		if err != nil {
			log.Fatalf("Error Reading: %v", err)
		}

		return nil, fmt.Errorf("failed to create container: %s", d)
	}

	connURL := "http://root:rootpassword@localhost:8529"
	if passwordless {
		connURL = "http://root@localhost:8529"
	}
	if params.Type == "" {
		params.Type = "arangodb"
	}

	if params.URL == "" {
		params.URL = connURL
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &ArangoDBContainer{
		ctr,
		connURL,
		driver,
	}, nil
}

// NewDriver helper function to create a new driver with the connection URL.
func (p *ArangoDBContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.URL == "" {
		params.URL = p.ConnURL
	}
	if params.Type == "" {
		params.Type = "arangodb"
	}
	return adapters.NewConnection(params)
}
