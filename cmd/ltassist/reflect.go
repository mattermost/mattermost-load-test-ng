package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func createStruct(v reflect.Value, dryRun bool) (reflect.Value, error) {
	val := reflect.New(v.Type()).Elem()
	for i := 0; i < v.Type().NumField(); i++ {
		f := v.Type().Field(i)
		if f.Type.Kind() == reflect.Struct {
			nv := reflect.New(f.Type).Elem()
			sv, err := createStruct(nv, dryRun)
			if err != nil {
				return val, err
			}
			val.Field(i).Set(sv)
			continue
		}
		if f.Type.Kind() == reflect.Slice {
			fmt.Printf("slices are not supported yet, skipping %q\n", f.Name)
			continue
		}
		fv, err := createField(f, dryRun)
		if err != nil {
			return val, err
		}
		val.Field(i).Set(fv)
	}
	return val, nil
}

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
		v.SetFloat(float64(f))
	case reflect.Float64:
		f, err := strconv.ParseFloat(data, 64)
		if err != nil {
			return v, err
		}
		v.SetFloat(f)
	}
	return v, nil
}

func createField(f reflect.StructField, dryRun bool) (reflect.Value, error) {
	doc, _ := findDoc(f.Name)
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
