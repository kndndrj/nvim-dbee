package testhelpers

import (
	"context"
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/adapters"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/gcloud"
	"github.com/testcontainers/testcontainers-go/wait"
)

// BigQueryContainer is a test container for BigQuery.
type BigQueryContainer struct {
	*gcloud.GCloudContainer
	ConnURL string
	Driver  *core.Connection
}

// NewBigQueryContainer creates a new BigQuery container with
// default adapter and connection. The params.URL is overwritten.
func NewBigQueryContainer(ctx context.Context, params *core.ConnectionParams) (*BigQueryContainer, error) {
	seedFile, err := GetTestDataFile("bigquery_seed.yaml")
	if err != nil {
		return nil, err
	}

	ctr, err := gcloud.RunBigQuery(
		ctx,
		"ghcr.io/goccy/bigquery-emulator:0.6.6",
		gcloud.WithProjectID("test-project"),
		gcloud.WithDataYAML(seedFile),
		tc.CustomizeRequest(tc.GenericContainerRequest{
			ProviderType: GetContainerProvider(),
			ContainerRequest: tc.ContainerRequest{
				ImagePlatform: "linux/amd64",
			},
		}),
		tc.WithWaitStrategy(wait.ForLog("[bigquery-emulator] gRPC")),
	)
	if err != nil {
		return nil, err
	}

	connURL := fmt.Sprintf("bigquery://%s?max-bytes-billed=1000&disable-query-cache=true&endpoint=%s", ctr.Settings.ProjectID, ctr.URI)
	if params.Type == "" {
		params.Type = "bigquery"
	}

	if params.URL == "" {
		params.URL = connURL
	}

	driver, err := adapters.NewConnection(params)
	if err != nil {
		return nil, err
	}

	return &BigQueryContainer{
		GCloudContainer: ctr,
		ConnURL:         connURL,
		Driver:          driver,
	}, nil
}

// NewDriver helper function to create a new driver with the connection URL.
func (p *BigQueryContainer) NewDriver(params *core.ConnectionParams) (*core.Connection, error) {
	if params.URL == "" {
		params.URL = p.ConnURL
	}
	if params.Type == "" {
		params.Type = "bigquery"
	}
	return adapters.NewConnection(params)
}
