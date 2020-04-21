package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/mattermost/mattermost-load-test-ng/logger"
)

// create a struct from a type, this function is called recursively so that
// we can walk on every field of it.
func createStruct(defaultValue interface{}, docPath string, dryRun bool) (reflect.Value, error) {
	v := reflect.Indirect(reflect.ValueOf(defaultValue))
	t := v.Type()

	switch defaultValue.(type) { // skip logging settings, and rely on defaults
	case logger.Settings:
		return v, nil
	}

	str := reflect.New(t).Elem()
	for i := 0; i < v.NumField(); i++ {
		defaultField := v.Field(i)
		switch defaultField.Type().Kind() {
		case reflect.Struct:
			dv := defaultField.Interface()
			newStruct, err := createStruct(dv, docPath, dryRun)
			if err != nil {
				return reflect.Zero(t), err
			}
			str.Field(i).Set(newStruct)
		case reflect.Slice:
			dv := defaultField.Interface()
			newSlice, err := createSlice(dv, docPath, dryRun)
			if err != nil {
				return reflect.Zero(t), err
			}
			str.Field(i).Set(newSlice)
		case reflect.Bool, reflect.Int, reflect.Float64, reflect.String:
			dvs := valueToString(defaultField)
			newField, err := createField(t.Field(i), docPath, dvs, dryRun)
			if err != nil {
				return reflect.Zero(t), err
			}
			str.Field(i).Set(newField)
		default:
			return reflect.Zero(t), fmt.Errorf("unimplemented struct field: %q", defaultField.Type().Name())
		}
	}
	return str, nil
}

// this function creates a slice for the given slice type
// does not support for some types (e.g. []int, []string etc.)
func createSlice(defaultValue interface{}, docPath string, dryRun bool) (reflect.Value, error) {
	t := reflect.ValueOf(defaultValue).Type().Elem()
	if dryRun {
		_, err := createStruct(reflect.New(t).Interface(), docPath, dryRun)
		return reflect.ValueOf(defaultValue), err
	}

	inp, err := readInput(fmt.Sprintf("size of %T", defaultValue), "1")
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	size, err := strconv.Atoi(strings.TrimSpace(inp))
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	values := reflect.Zero(reflect.SliceOf(t))
	for i := 0; i < size; i++ {
		dv := reflect.New(t).Interface()
		s, err := createStruct(dv, docPath, dryRun)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		values = reflect.Append(values, s)
	}
	return values, nil
}

// this is a glue function to find doc, get user input and assign to a value
func createField(f reflect.StructField, docPath, defaultValue string, dryRun bool) (reflect.Value, error) {
	doc, err := findDoc(f.Name, docPath)
	if err != nil {
		return reflect.Zero(f.Type), err
	}
	if dryRun && (f.Type.Kind().String() != doc.dataType || doc.text == "") {
		return reflect.ValueOf(nil), fmt.Errorf("check the docs for %q data type: %q", f.Name, doc.dataType)
	} else if dryRun {
		return reflect.Zero(f.Type), nil
	}
	// print the name and doc
	fmt.Println(color.GreenString(f.Name))
	fmt.Println(doc.text)
	if defaultValue != "" { // display default value, if there is any
		fmt.Printf("Default: %s\n", color.GreenString(defaultValue))
	}
	for {
		inp, err := readInput(f.Type.Kind().String(), defaultValue)
		if err != nil {
			return reflect.Zero(f.Type), err
		}
		v, err := setValue(f.Type, inp)
		if err != nil {
			fmt.Println(color.RedString("invalid type. Retry:"))
			continue
		}
		return v, nil
	}
}

// converts given string into reflect.Value, the value is assignable to struct
// field.
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

func valueToString(v reflect.Value) string {
	switch v.Type().Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Float64:
		return fmt.Sprintf("%f", v.Float())
	default:
		return ""
	}
}
