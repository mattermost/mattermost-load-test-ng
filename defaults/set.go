package defaults

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Set(value interface{}) error {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("value should be a pointer")
	}
	_, err := toDefaults(value)
	if err != nil {
		return err
	}
	return nil
}

// toDefaults assigns default values of a struct
func toDefaults(defaultValue interface{}) (reflect.Value, error) {
	v := reflect.Indirect(reflect.ValueOf(defaultValue))
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		switch field.Type().Kind() {
		case reflect.Struct:
			dv := field.Addr().Interface()
			v, err := toDefaults(dv)
			if err != nil {
				return reflect.ValueOf(nil), err
			}
			field.Set(v)
		case reflect.Slice:
			tag, ok := t.Field(i).Tag.Lookup("default_size")
			if !ok {
				continue
			}
			size, err := strconv.Atoi(tag)
			if err != nil {
				return reflect.ValueOf(nil), fmt.Errorf("invalid size definition: %q", tag)
			}
			dv := field.Interface()
			newSlice, err := createSlice(dv, size)
			if err != nil {
				return reflect.Zero(t), err
			}
			field.Set(newSlice)
		case reflect.Bool, reflect.Int, reflect.Float64, reflect.String:
			tag, ok := t.Field(i).Tag.Lookup("default")
			if !ok {
				continue
			}
			def, err := setValue(field.Type(), tag)
			if err != nil {
				return reflect.ValueOf(nil), fmt.Errorf("could not set value: %w", err)
			}
			field.Set(def)
		default:
			return reflect.ValueOf(nil), fmt.Errorf("unimplemented struct field: %s", t.Field(i).Type.Kind())
		}
	}
	return v, nil
}

// converts given string into reflect.Value, the value is assignable to a
// struct field.
func setValue(t reflect.Type, data string) (reflect.Value, error) {
	data = strings.TrimSpace(data)
	v := reflect.New(t).Elem()
	switch t.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(data)
		if err != nil {
			return reflect.Zero(t), err
		}
		v.SetBool(b)
	case reflect.String:
		v.SetString(data)
	case reflect.Int:
		i, err := strconv.Atoi(data)
		if err != nil {
			return reflect.Zero(t), err
		}
		v.SetInt(int64(i))
	case reflect.Float64:
		f, err := strconv.ParseFloat(data, 64)
		if err != nil {
			return reflect.Zero(t), err
		}
		v.SetFloat(f)
	}
	return v, nil
}

// this function creates a slice for the given slice type
// does not support for some types (e.g. []int, []string etc.)
func createSlice(defaultValue interface{}, size int) (reflect.Value, error) {
	t := reflect.ValueOf(defaultValue).Type().Elem()
	values := reflect.Zero(reflect.SliceOf(t))
	for i := 0; i < size; i++ {
		dv := reflect.New(t).Interface()
		s, err := toDefaults(dv)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		values = reflect.Append(values, s)
	}
	return values, nil
}
