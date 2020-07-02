package defaults

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	rangeRegex = regexp.MustCompile(`range:(\[|\()(\S*)\,(\S*)(\]|\))`)
	oneofRegex = regexp.MustCompile(`oneof:(\{)(.*)(\})`)
)

// Validate validates each field of the value
func Validate(value interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(value))
	t := v.Type()

	m := reflect.ValueOf(value).MethodByName("IsValid")
	if m.IsValid() {
		e := m.Call([]reflect.Value{})
		err, ok := e[0].Interface().(error)
		if ok && err != nil {
			return err
		}
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		switch field.Type().Kind() {
		case reflect.Struct:
			dv := field.Interface()
			if err := Validate(dv); err != nil {
				return err
			}
		case reflect.Slice, reflect.Map:
			dv := reflect.ValueOf(field.Interface())
			for j := 0; j < dv.Len(); j++ {
				if err := Validate(dv.Index(j).Interface()); err != nil {
					return err
				}
			}
		case reflect.Bool, reflect.Int, reflect.Int64, reflect.Float64, reflect.String:
			tag, ok := t.Field(i).Tag.Lookup("validate")
			if !ok {
				continue
			}
			if err := validate(tag, t.Field(i).Name, v, v.Field(i)); err != nil {
				return err
			}
		case reflect.Chan:
			return nil
		default:
			return fmt.Errorf("unimplemented struct field type: %s", t.Field(i).Name)
		}
	}
	return nil
}

func validate(validation, fieldName string, p, v reflect.Value) error {
	switch validation {
	case "url":
		s := v.String()
		_, err := url.ParseRequestURI(s)
		if err != nil {
			return err
		}
	case "email":
		s := v.String()
		if !emailRegex.MatchString(s) {
			return fmt.Errorf("%s is not a valid e-mail address", s)
		}
	case "notempty":
		switch v.Type().Kind() {
		case reflect.String:
			s := v.String()
			if s == "" {
				return fmt.Errorf("%s is empty", fieldName)
			}
		}
	case "alpha":
		s := v.String()
		if s == "" || !isAlphanumeric(s) {
			return fmt.Errorf("%s is not alphanumeric", fieldName)
		}
	case "file":
		s := v.String()
		if _, err := os.Stat(s); os.IsNotExist(err) {
			return fmt.Errorf("%s: %w", fieldName, err)
		}
	default:
		if strings.HasPrefix(validation, "range") {
			if !rangeRegex.MatchString(validation) {
				return errors.New("invalid range declaration")
			}
			matches := rangeRegex.FindStringSubmatch(validation)
			mins, err := validateFromField(p, matches[2])
			if err != nil {
				return err
			}
			maxs, err := validateFromField(p, matches[3])
			if err != nil {
				return err
			}
			if err := validateFromRange(v, mins, maxs, matches[1], matches[4]); err != nil {
				return fmt.Errorf("%s is not in the range of %s: %w", fieldName, validation, err)
			}
		} else if strings.HasPrefix(validation, "oneof") {
			if !oneofRegex.MatchString(validation) {
				return errors.New("ivalid oneof declaration")
			}
			valids := oneofRegex.FindStringSubmatch(validation)[2]
			if err := validateFromOneofValues(v, strings.Split(valids, ",")); err != nil {
				return fmt.Errorf("%s is not valid: %w", fieldName, err)
			}
		}
	}
	return nil
}

func validateFromRange(value reflect.Value, mins, maxs, minInterval, maxInterval string) error {
	var min, max, val float64
	var err error
	switch value.Type().Kind() {
	case reflect.Int, reflect.Int64:
		if mins != "" {
			min, err = strconv.ParseFloat(mins, 64)
		}
		if maxs != "" {
			max, err = strconv.ParseFloat(maxs, 64)
		}
		val = float64(value.Int())
	case reflect.Float64:
		if mins != "" {
			min, err = strconv.ParseFloat(mins, 64)
		}
		if maxs != "" {
			max, err = strconv.ParseFloat(maxs, 64)
		}
		val = value.Float()
	default:
		return errors.New("could not validate this value within a range")
	}
	if err != nil {
		return err
	}

	if mins != "" {
		if minInterval == "(" {
			if min >= val {
				return errors.New("value is lesser or equal")
			}
		} else {
			if min > val {
				return errors.New("value is lesser")
			}
		}
	}

	if maxs != "" {
		if maxInterval == ")" {
			if max <= val {
				return errors.New("value is greater or equal")
			}
		} else {
			if max < val {
				return errors.New("value is greater")
			}
		}
	}
	return nil
}

func validateFromField(value reflect.Value, valuestr string) (string, error) {
	if len(valuestr) > 0 && valuestr[0] == '$' {
		var v reflect.Value
		var found bool
		for i := 0; i < value.Type().NumField(); i++ {
			name := value.Type().Field(i).Name
			if name == valuestr[1:] {
				v = value.Field(i)
				found = true
			}
		}
		if !found {
			return "", fmt.Errorf("%q has no field %q", value.Type(), valuestr[1:])
		}

		switch v.Type().Kind() {
		case reflect.Int, reflect.Int64:
			return fmt.Sprintf("%d", v.Int()), nil
		case reflect.Float64:
			return fmt.Sprintf("%f", v.Float()), nil
		default:
			return "", fmt.Errorf("%q is not a supported field type for using as parameter", v.Type().Kind())
		}
	}
	return valuestr, nil
}

func validateFromOneofValues(value reflect.Value, values []string) error {
	switch value.Kind() {
	case reflect.String:
		s := value.String()
		for _, str := range values {
			if s == strings.TrimSpace(str) {
				return nil
			}
		}
	case reflect.Int:
		d := value.Int()
		for _, str := range values {
			n, err := strconv.Atoi(strings.TrimSpace(str))
			if err != nil {
				return err
			}
			if d == int64(n) {
				return nil
			}
		}
	case reflect.Float64:
		f := value.Float()
		for _, str := range values {
			n, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
			if err != nil {
				return err
			}
			if f == n {
				return nil
			}
		}
	default:
		return errors.New("unsupported field type for oneof validation")
	}
	return errors.New("value is not one of valid values")
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}
