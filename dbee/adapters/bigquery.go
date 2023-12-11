package adapters

import (
	"context"
	"fmt"
	"net/url"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"

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
// Where:
//   - "project" is optional. If not set, the project will attempt to be
//     detected from the credentials and current gcloud settings.
//   - "options" is a ampersand-separated list of key=value arguments.
//
// The supported "options" are:
//   - credentials=path/to/creds/file.json
//   - disable-cache=true|false
//   - max-bytes-billed=integer
//   - enable-storage-read=true|false
//   - use-legacy-sql=true|false
//   - location=google-cloud-location
//
// If credentials are not explicitly specified, credentials will attempt
// to be located according to the Google Default Credentials process.
func (bq *BigQuery) Connect(rawURL string) (core.Driver, error) {
	ctx := context.TODO()

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

	options := []option.ClientOption{
		option.WithTelemetryDisabled(),
	}

	params := u.Query()
	_ = callIfStringSet("credentials", params, func(file string) error {
		options = append(options, option.WithCredentialsFile(file))
		return nil
	})

	bqc, err := bigquery.NewClient(ctx, u.Host, options...)
	if err != nil {
		return nil, err
	}

	client := &bigQueryDriver{
		c: bqc,
	}

	_ = setStringOption(&client.location, "location", params)

	if err := setInt64Option(&client.maxBytesBilled, "max-bytes-billed", params); err != nil {
		return nil, err
	}

	if err := setBoolOption(&client.disableQueryCache, "disable-cache", params); err != nil {
		return nil, err
	}

	if err := setBoolOption(&client.useLegacySQL, "use-legacy-sql", params); err != nil {
		return nil, err
	}

	if err := callIfBoolSet("enable-storage-read", params, func() error {
		return client.c.EnableStorageReadClient(ctx, options...)
	}, nil); err != nil {
		return nil, err
	}

	return client, nil
}

func (*BigQuery) GetHelpers(opts *core.HelperOptions) map[string]string {
	return map[string]string{
		"List":    fmt.Sprintf("SELECT * FROM `%s` LIMIT 500", opts.Table),
		"Columns": fmt.Sprintf("SELECT * FROM `%s.INFORMATION_SCHEMA.COLUMNS` WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'", opts.Schema, opts.Schema, opts.Table),
	}
}
