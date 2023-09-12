package vim

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	ErrNotAStruct          = func(v any) error { return fmt.Errorf("provided type is not a struct: %T", v) }
	ErrRequiredFieldNotSet = func(field string) error { return fmt.Errorf("required field %q not set", field) }
	ErrInvalidFieldType    = func(field string, want any, got any) error {
		return fmt.Errorf("value for field %q not of compatible type: want %T, got %T", field, want, got)
	}
)

// FuncArgs wraps nvim function arguments into a declarative structure
type FuncArgs[T any] struct {
	args map[string]any `msgpack:",array"`
}

func (fa *FuncArgs[T]) Set(m map[string]any)  {
	fa.args = m
}

func (fa *FuncArgs[T]) Parse() (*T, error) {
	t := new(T)

	if reflect.ValueOf(*t).Kind() != reflect.Struct {
		return new(T), ErrNotAStruct(*t)
	}

	v := reflect.ValueOf(t).Elem()

	for i := 0; i < v.NumField(); i += 1 {
		valueField := v.Field(i)
		typeField := v.Type().Field(i)

		// parse tags
		tags := strings.Split(typeField.Tag.Get("arg"), ",")
		name := typeField.Name
		required := true
		for i, tag := range tags {
			if i == 0 && tag != "" {
				name = tag
			} else if tag == "optional" {
				required = false
			}
		}

		if !valueField.CanSet() {
			if required {
				return new(T), ErrRequiredFieldNotSet(name)
			}
			continue
		}

		val, ok := fa.args[name]
		if !ok {
			if required {
				return new(T), ErrRequiredFieldNotSet(name)
			}
			continue
		}

		// convert int variants (int32, uint64...) to int
		if typeField.Type.Kind() == reflect.Int {
			val = toInt(val)
		}

		setVal := reflect.ValueOf(val)
		if setVal.Type().Name() != typeField.Type.Name() {
			return new(T), ErrInvalidFieldType(name, valueField.Interface(), val)
		}

		valueField.Set(setVal)
	}

	return t, nil
}

func toInt(v any) any {
	switch t := v.(type) {
	case int8:
		return int(t)
	case int16:
		return int(t)
	case int32:
		return int(t)
	case int64:
		return int(t)
	case uint:
		return int(t)
	case uint8:
		return int(t)
	case uint16:
		return int(t)
	case uint32:
		return int(t)
	case uint64:
		return int(t)
	}

	return v
}
