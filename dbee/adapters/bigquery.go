package adapters

import (
	"context"
	"fmt"
	"net/url"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
	"google.golang.org/api/option/internaloption"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kndndrj/nvim-dbee/dbee/core"
)

// Register client
func init() {
	_ = register(&BigQuery{}, "bigquery")
}

var _ core.Adapter = (*BigQuery)(nil)

type BigQuery struct{}

// Connect creates a [BigQuery] client connected to the project specified
// in the url. The format of the url is as follows:
//
//	bigquery://[project][?options]
//
// where project is optional. If not set, the project will attempt to be
// detected from the credentials and current gcloud settings.
//
// The options query parameters map directly to [bigquery.QueryConfig] fields
// using kebab-case. For example, MaxBytesBilled becomes max-bytes-billed.
//
// Common options include:
//   - credentials=path/to/creds.json: Path to credentials file
//   - max-bytes-billed=integer: Maximum bytes to be billed
//   - disable-query-cache=bool: Whether to disable query cache
//   - use-legacy-sql=bool: Whether to use legacy SQL
//   - location=string: Query location
//   - enable-storage-read=bool: Enable BigQuery Storage API
//
// For internal testing:
//   - endpoint=url: Custom endpoint for test containers
//
// If credentials are not specified, they will be located according to
// the Google Default Credentials process.
func (bq *BigQuery) Connect(rawURL string) (core.Driver, error) {
	ctx := context.Background()

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "bigquery" {
		return nil, fmt.Errorf("unexpected scheme: %q", u.Scheme)
	}

	if u.Host == "" {
		u.Host = bigquery.DetectProjectID
	}

	options := []option.ClientOption{option.WithTelemetryDisabled()}
	params := u.Query()

	// special param to indicate we are running in testcontainer.
	if endpoint := params.Get("endpoint"); endpoint != "" {
		options = append(options,
			option.WithEndpoint(endpoint),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			option.WithoutAuthentication(),
			internaloption.SkipDialSettingsValidation(),
		)
	} else {
		callIfStringSet("credentials", params, func(file string) error {
			options = append(options, option.WithCredentialsFile(file))
			return nil
		})
	}

	bqc, err := bigquery.NewClient(ctx, u.Host, options...)
	if err != nil {
		return nil, err
	}

	client := &bigQueryDriver{c: bqc}
	if err = setQueryConfigFromParams(&client.QueryConfig, params); err != nil {
		return nil, err
	}

	if err = callIfBoolSet("enable-storage-read", params, func() error {
		return client.c.EnableStorageReadClient(ctx, options...)
	}, nil); err != nil {
		return nil, err
	}

	return client, nil
}

func (*BigQuery) GetHelpers(opts *core.TableOptions) map[string]string {
	return map[string]string{
		"List":    fmt.Sprintf("SELECT * FROM `%s` TABLESAMPLE SYSTEM (5 PERCENT)", opts.Table),
		"Columns": fmt.Sprintf("SELECT * FROM `%s.INFORMATION_SCHEMA.COLUMNS` WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'", opts.Schema, opts.Schema, opts.Table),
	}
}
