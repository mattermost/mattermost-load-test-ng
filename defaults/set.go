package defaults

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Set sets the default values to fields
func Set(value interface{}) error {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("value should be a pointer")
	}
	return structDefaults(value)
}

// structDefaults assigns default values of a struct
func structDefaults(value interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(value))
	t := v.Type()

	if v.Kind() != reflect.Struct {
		return errors.New("value should be struct type")
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			return nil
		}

		switch field.Type().Kind() {
		case reflect.Struct:
			dv := field.Addr().Interface()
			err := structDefaults(dv)
			if err != nil {
				return err
			}
			field.Set(reflect.Indirect(reflect.ValueOf(dv)))
		case reflect.Slice:
			tag, ok := t.Field(i).Tag.Lookup("default_size")
			if !ok {
				continue
			}
			size, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid size definition: %q", tag)
			}
			dv := field.Interface()
			newSlice, err := createSlice(dv, size)
			if err != nil {
				return err
			}
			field.Set(newSlice)
		case reflect.Chan:
			tag, ok := t.Field(i).Tag.Lookup("default_size")
			if !ok {
				continue
			}
			size, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid size definition: %q", tag)
			}
			ch := reflect.MakeChan(field.Type(), size)
			field.Set(ch)
		case reflect.Map:
			tag, ok := t.Field(i).Tag.Lookup("default_size")
			if !ok {
				continue
			}
			size, err := strconv.Atoi(tag)
			if err != nil {
				return fmt.Errorf("invalid size definition: %q", tag)
			}
			dv := field.Interface()
			m, err := createMap(dv, size)
			if err != nil {
				return fmt.Errorf("could not create map: %w", err)
			}
			field.Set(m)
		case reflect.Bool, reflect.Int, reflect.Float64, reflect.String:
			tag, ok := t.Field(i).Tag.Lookup("default")
			if !ok {
				continue
			}
			def, err := setValue(field.Type(), tag)
			if err != nil {
				return fmt.Errorf("could not set value: %w", err)
			}
			field.Set(def)
		default:
			return fmt.Errorf("unimplemented struct field type: %s", t.Field(i).Type.Kind())
		}
	}
	return nil
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
func createSlice(defaultValue interface{}, size int) (reflect.Value, error) {
	t := reflect.ValueOf(defaultValue).Type().Elem()
	if t.Kind() == reflect.Struct {
		values := reflect.Zero(reflect.SliceOf(t))
		for i := 0; i < size; i++ {
			dv := reflect.New(t).Interface()
			err := structDefaults(dv)
			if err != nil {
				return reflect.ValueOf(nil), err
			}
			values = reflect.Append(values, reflect.Indirect(reflect.ValueOf(dv)))
		}
		return values, nil
	}
	return reflect.MakeSlice(t, size, size), nil
}

// this function creates a map for the given map type
func createMap(defaultValue interface{}, size int) (reflect.Value, error) {
	t := reflect.ValueOf(defaultValue).Type()
	if t.Elem().Kind() == reflect.Struct {
		values := reflect.Zero(reflect.MapOf(t.Key(), t.Elem()))
		for i := 0; i < size; i++ {
			dv := reflect.New(t).Interface()
			err := structDefaults(dv)
			if err != nil {
				return reflect.ValueOf(nil), err
			}
			values = reflect.Append(values, reflect.Indirect(reflect.ValueOf(dv)))
		}
		return values, nil
	}
	return reflect.MakeMapWithSize(t, size), nil
}
