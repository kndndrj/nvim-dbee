package adapters

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"cloud.google.com/go/bigquery"
	"github.com/kndndrj/nvim-dbee/dbee/core"
	"github.com/kndndrj/nvim-dbee/dbee/core/builders"
	"google.golang.org/api/iterator"
)

var _ core.Driver = (*bigQueryDriver)(nil)

type bigQueryDriver struct {
	c *bigquery.Client
	bigquery.QueryConfig
}

func (d *bigQueryDriver) Query(ctx context.Context, queryStr string) (core.ResultStream, error) {
	query := d.c.Query(queryStr)
	d.Q = query.Q
	query.QueryConfig = d.QueryConfig

	iter, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	var currentRow bigqueryRowLoader
	hasNext := true

	// schema and header only detectable after retrieving the first reslt
	if err := iter.Next(&currentRow); err != nil {
		if errors.Is(err, iterator.Done) {
			hasNext = false
		} else {
			return nil, err
		}
	}

	header := d.buildHeader("", iter.Schema)

	nextFn := func() (core.Row, error) {
		if !hasNext {
			return nil, nil
		}

		row := currentRow.row
		var nextLoader bigqueryRowLoader
		if err := iter.Next(&nextLoader); err != nil {
			if errors.Is(err, iterator.Done) {
				hasNext = false
				return row, nil
			}
			return nil, err
		}
		currentRow = nextLoader
		return row, nil
	}
	hasNextFn := func() bool { return hasNext }

	result := builders.NewResultStreamBuilder().
		WithNextFunc(nextFn, hasNextFn).
		WithHeader(header).
		Build()
	return result, nil
}

func (d *bigQueryDriver) Columns(opts *core.TableOptions) ([]*core.Column, error) {
	query := fmt.Sprintf(
		"SELECT COLUMN_NAME, DATA_TYPE FROM `%s.INFORMATION_SCHEMA.COLUMNS` WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'",
		opts.Schema, opts.Schema, opts.Table)

	result, err := d.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return builders.ColumnsFromResultStream(result)
}

func (d *bigQueryDriver) Structure() (layouts []*core.Structure, err error) {
	ctx := context.Background()

	datasetsIter := d.c.Datasets(ctx)
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

func (d *bigQueryDriver) Close() { _ = d.c.Close() }

func (d *bigQueryDriver) buildHeader(parentName string, schema bigquery.Schema) (columns core.Header) {
	for _, field := range schema {
		if field.Type == bigquery.RecordFieldType {
			nestedName := field.Name
			if parentName != "" {
				nestedName = parentName + "." + nestedName
			}
			columns = append(columns, d.buildHeader(nestedName, field.Schema)...)
		} else {
			columns = append(columns, field.Name)
		}
	}

	return columns
}

type bigqueryRowLoader struct{ row core.Row }

func (l *bigqueryRowLoader) Load(row []bigquery.Value, schema bigquery.Schema) error {
	l.row = make(core.Row, len(row))

	for i, col := range row {
		l.row[i] = col
	}

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

func setQueryConfigFromParams(config *bigquery.QueryConfig, params url.Values) error {
	v := reflect.ValueOf(config).Elem()
	t := v.Type()

	// NumField panics if it isn't a struct
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %v", t.Kind())
	}

	for i := 0; i < t.NumField(); i++ {
		field, fieldValue := t.Field(i), v.Field(i)

		paramName := toKebabCase(field.Name)
		if val := params.Get(paramName); val != "" {
			return setFieldFromString(fieldValue, val)
		}
	}
	return nil
}

// toKebabCase converts a string to kebab-case
func toKebabCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if (unicode.IsUpper(r)) && (i != 0 &&
			!unicode.IsUpper(rune(s[i-1]))) && i+1 != len(s) {
			result.WriteByte('-')
		}
		result.WriteString(strings.ToLower(string(r)))
	}
	return result.String()
}

// setFieldFromString sets a field value from its string representation
func setFieldFromString(fieldValue reflect.Value, val string) error {
	if !fieldValue.CanSet() {
		return fmt.Errorf("field is not settable")
	}

	if val == "" {
		return nil
	}

	switch fieldValue.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("failed to parse bool: %w", err)
		}
		fieldValue.SetBool(b)
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse int: %w", err)
		}
		fieldValue.SetInt(i)
	case reflect.String:
		fieldValue.SetString(val)
	case reflect.Ptr:
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		}
		return setFieldFromString(fieldValue.Elem(), val)

	default:
		return fmt.Errorf("unsupported field type %s", fieldValue.Kind())
	}
	return nil
}
