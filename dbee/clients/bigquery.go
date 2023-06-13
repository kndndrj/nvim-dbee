package clients

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/kndndrj/nvim-dbee/dbee/clients/common"
	"github.com/kndndrj/nvim-dbee/dbee/conn"
	"github.com/kndndrj/nvim-dbee/dbee/models"
)

// Register client
func init() {
	c := func(url string) (conn.Client, error) {
		return NewBigQuery(url)
	}
	_ = Store.Register("bigquery", c)
}

type BigQueryClient struct {
	c                 *bigquery.Client
	location          string
	maxBytesBilled    int64
	disableQueryCache bool
	useLegacySQL      bool
}

// NewBigQuery creates a [BigQueryClient] connected to the project specified
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
func NewBigQuery(rawURL string) (*BigQueryClient, error) {
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

	client := &BigQueryClient{
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

func (c *BigQueryClient) Query(queryStr string) (models.IterResult, error) {
	query := c.c.Query(queryStr)
	query.DisableQueryCache = c.disableQueryCache
	query.MaxBytesBilled = c.maxBytesBilled
	query.UseLegacySQL = c.useLegacySQL
	query.Location = c.location

	iter, err := query.Read(context.TODO())
	if err != nil {
		return nil, err
	}

	// schema isn't available until the first call to iter.Next()
	var firstRowLoader bigqueryRowLoader
	if err := iter.Next(&firstRowLoader); err != nil {
		return nil, err
	}

	header := c.buildHeader("", iter.Schema)

	result := common.NewResultBuilder().
		WithNextFunc(func() (models.Row, error) {
			if firstRowLoader.row != nil {
				row := firstRowLoader.row
				firstRowLoader.row = nil
				return row, nil
			}

			var loader bigqueryRowLoader
			if err := iter.Next(&loader); err != nil {
				if errors.Is(err, iterator.Done) {
					return nil, nil
				}

				return nil, err
			}

			return loader.row, nil
		}).
		WithHeader(header).
		WithCloseFunc(func() { /* noop */ }).
		WithMeta(models.Meta{
			Query:     queryStr,
			Timestamp: time.Now(),
		}).
		Build()
	return result, nil
}

func (c *BigQueryClient) Layout() (layouts []models.Layout, err error) {
	ctx := context.TODO()

	datasetsIter := c.c.Datasets(ctx)
	for {
		dataset, err := datasetsIter.Next()
		if err != nil {
			if !errors.Is(err, iterator.Done) {
				return nil, err
			}

			break
		}

		datasetLayout := models.Layout{
			Name:     dataset.DatasetID,
			Schema:   dataset.DatasetID,
			Database: dataset.ProjectID,
			Type:     models.LayoutNone,
			Children: []models.Layout{},
		}

		tablesIter := dataset.Tables(ctx)
		for {
			table, err := tablesIter.Next()
			if err != nil {
				if !errors.Is(err, iterator.Done) {
					return nil, err
				}

				break
			}

			datasetLayout.Children = append(datasetLayout.Children, models.Layout{
				Name:     table.TableID,
				Schema:   table.DatasetID,
				Database: table.ProjectID,
				Type:     models.LayoutTable,
				Children: nil,
			})
		}

		layouts = append(layouts, datasetLayout)
	}

	return layouts, nil
}

func (c *BigQueryClient) Close() {
	_ = c.c.Close()
}

func (c *BigQueryClient) buildHeader(parentName string, schema bigquery.Schema) (columns models.Header) {
	for _, field := range schema {
		if field.Type == bigquery.RecordFieldType {
			nestedName := field.Name
			if parentName != "" {
				nestedName = parentName + "." + nestedName
			}
			columns = append(columns, c.buildHeader(nestedName, field.Schema)...)
		} else {
			columns = append(columns, field.Name)
		}
	}

	return columns
}

type bigqueryRowLoader struct {
	row models.Row
}

func (l *bigqueryRowLoader) Load(row []bigquery.Value, schema bigquery.Schema) error {
	l.row = make(models.Row, len(row))

	for i, col := range row {
		l.row[i] = col
	}

	return nil
}

func setBoolOption(field *bool, name string, params url.Values) error {
	return setOption(field, name, params, strconv.ParseBool)
}

func setInt64Option(field *int64, name string, params url.Values) error {
	return setOption(field, name, params, func(s string) (int64, error) {
		return strconv.ParseInt(s, 10, 64)
	})
}

func setStringOption(field *string, name string, params url.Values) error {
	return setOption(field, name, params, func(s string) (string, error) { return s, nil })
}

func setOption[T any](field *T, name string, params url.Values, parse func(string) (T, error)) error {
	setting := params.Get(name)
	if setting == "" {
		return nil
	}

	val, err := parse(setting)
	if err != nil {
		return fmt.Errorf("invalid value for %q: %w", name, err)
	}

	*field = val
	return nil
}

func callIfBoolSet(name string, params url.Values, onTrue, onFalse func() error) error {
	if onTrue == nil {
		onTrue = func() error { return nil }
	}
	if onFalse == nil {
		onFalse = func() error { return nil }
	}

	return callIfSet(name, params, strconv.ParseBool, func(b bool) error {
		if b {
			return onTrue()
		}
		return onFalse()
	})
}

func callIfStringSet(name string, params url.Values, onSet func(string) error) error {
	return callIfSet(name, params, func(s string) (string, error) { return s, nil }, onSet)
}

func callIfSet[T any](name string, params url.Values, parse func(string) (T, error), cb func(T) error) error {
	setting := params.Get(name)
	if setting == "" {
		return nil
	}

	val, err := parse(setting)
	if err != nil {
		return fmt.Errorf("invalid value for %q: %w", name, err)
	}

	return cb(val)
}
