package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// create a struct from its value, this function is called recursively so that
// we can walk on every value for it.
func createStruct(v reflect.Value, docPath string, dryRun bool) (reflect.Value, error) {
	val := reflect.New(v.Type()).Elem()
	for i := 0; i < v.Type().NumField(); i++ {
		f := v.Type().Field(i)
		if f.Type.Kind() == reflect.Struct {
			nv := reflect.New(f.Type).Elem()
			sv, err := createStruct(nv, docPath, dryRun)
			if err != nil {
				return val, err
			}
			val.Field(i).Set(sv)
			continue
		}
		if f.Type.Kind() == reflect.Slice {
			values, err := createSlice(f.Type.Elem(), docPath, dryRun)
			if err != nil {
				return val, err
			}
			val.Field(i).Set(values)
			continue
		}
		fv, err := createField(f, docPath, dryRun)
		if err != nil {
			return val, err
		}
		val.Field(i).Set(fv)
	}
	return val, nil
}

// this function creates a slice for the given type
// TODO: add support for primitive types (e.g. []int, []string etc.)
func createSlice(t reflect.Type, docPath string, dryRun bool) (reflect.Value, error) {
	if dryRun {
		_, err := createStruct(reflect.Zero(t), docPath, dryRun)
		return reflect.MakeSlice(reflect.SliceOf(t), 0, 0), err
	}
	inp := readInput(fmt.Sprintf("Enter the size of []%s", t), "integer")
	size, err := strconv.Atoi(strings.TrimSpace(inp))
	if err != nil {
		panic(err)
	}
	values := reflect.Zero(reflect.SliceOf(t))
	for i := 0; i < size; i++ {
		nv := reflect.Zero(t)
		v, err := createStruct(nv, docPath, dryRun)
		if err != nil {
			panic(err)
		}
		values = reflect.Append(values, v)
	}
	return values, nil
}

// converts given string into reflect.Value, the value is assignable to struct
// field.
func toValue(data string, t reflect.Type) (reflect.Value, error) {
	v := reflect.New(t).Elem()
	switch t.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(data)
		if err != nil {
			return v, err
		}
		v.SetBool(b)
	case reflect.String:
		v.SetString(data)
	case reflect.Int:
		i, err := strconv.Atoi(data)
		if err != nil {
			return v, err
		}
		v.SetInt(int64(i))
	case reflect.Float32:
		f, err := strconv.ParseFloat(data, 32)
		if err != nil {
			return v, err
		}
		v.SetFloat(f)
	case reflect.Float64:
		f, err := strconv.ParseFloat(data, 64)
		if err != nil {
			return v, err
		}
		v.SetFloat(f)
	}
	return v, nil
}

// this is a glue function to find doc, get user input and assign to a value
func createField(f reflect.StructField, docPath string, dryRun bool) (reflect.Value, error) {
	doc, err := findDoc(f.Name, docPath)
	if err != nil {
		return reflect.Zero(f.Type), err
	}
	if dryRun && (f.Type.Kind().String() != doc.dataType || doc.text == "") {
		return reflect.Zero(f.Type), fmt.Errorf("check the docs for %q", f.Name)
	} else if dryRun {
		return reflect.Zero(f.Type), nil
	}
	// print the doc
	text := adjustToWindow(doc.text)
	fmt.Println(text)
	lines := strings.Count(text, "\n")
	for {
		inp := readInput(f.Name, f.Type.Kind().String())
		v, err := toValue(strings.TrimSpace(inp), f.Type)
		// TODO: add skip
		if err != nil {
			fmt.Println("invalid type. Retry:")
			lines += 2 // we added 2 new lines for reading the the input
			continue
		}
		rewind(lines + 2)
		return v, nil
	}
}
