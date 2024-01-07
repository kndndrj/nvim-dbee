package adapters

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"cloud.google.com/go/bigquery"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	"google.golang.org/api/iterator"
)

var _ core.Driver = (*bigQueryDriver)(nil)

type bigQueryDriver struct {
	c                 *bigquery.Client
	location          string
	maxBytesBilled    int64
	disableQueryCache bool
	useLegacySQL      bool
}

func (c *bigQueryDriver) Query(ctx context.Context, queryStr string) (core.ResultStream, error) {
	query := c.c.Query(queryStr)
	query.DisableQueryCache = c.disableQueryCache
	query.MaxBytesBilled = c.maxBytesBilled
	query.UseLegacySQL = c.useLegacySQL
	query.Location = c.location

	iter, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	// schema isn't available until the first call to iter.Next()
	var firstRowLoader bigqueryRowLoader
	if err := iter.Next(&firstRowLoader); err != nil {
		return nil, err
	}

	header := c.buildHeader("", iter.Schema)

	hasNext := true
	nextFn := func() (core.Row, error) {
		if firstRowLoader.row != nil {
			row := firstRowLoader.row
			firstRowLoader.row = nil
			return row, nil
		}

		var loader bigqueryRowLoader
		if err := iter.Next(&loader); err != nil {
			if errors.Is(err, iterator.Done) {
				hasNext = false
			}

			return nil, err
		}

		return loader.row, nil
	}

	hasNextFn := func() bool {
		return hasNext
	}

	result := builders.NewResultStreamBuilder().
		WithNextFunc(nextFn, hasNextFn).
		WithHeader(header).
		Build()
	return result, nil
}

// TODO(ms):
func (c *bigQueryDriver) Columns(opts *core.HelperOptions) ([]*core.Columns, error) {
	return nil, nil
}

func (c *bigQueryDriver) Structure() (layouts []*core.Structure, err error) {
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

		datasetLayout := &core.Structure{
			Name:     dataset.DatasetID,
			Schema:   dataset.DatasetID,
			Type:     core.StructureTypeNone,
			Children: []*core.Structure{},
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

			datasetLayout.Children = append(datasetLayout.Children, &core.Structure{
				Name:     table.TableID,
				Schema:   table.DatasetID,
				Type:     core.StructureTypeTable,
				Children: nil,
			})
		}

		layouts = append(layouts, datasetLayout)
	}

	return layouts, nil
}

func (c *bigQueryDriver) Close() {
	_ = c.c.Close()
}

func (c *bigQueryDriver) buildHeader(parentName string, schema bigquery.Schema) (columns core.Header) {
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
	row core.Row
}

func (l *bigqueryRowLoader) Load(row []bigquery.Value, schema bigquery.Schema) error {
	l.row = make(core.Row, len(row))

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
