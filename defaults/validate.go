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

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		switch field.Type().Kind() {
		case reflect.Struct:
			dv := field.Interface()
			if err := Validate(dv); err != nil {
				return err
			}
		case reflect.Slice:
			dv := reflect.ValueOf(field.Interface())
			for j := 0; j < dv.Len(); j++ {
				if err := Validate(dv.Index(j).Interface()); err != nil {
					return err
				}
			}
		case reflect.Bool, reflect.Int, reflect.Float64, reflect.String:
			tag, ok := t.Field(i).Tag.Lookup("validate")
			if !ok {
				continue
			}
			if err := validate(tag, t.Field(i).Name, v, v.Field(i)); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unimplemented struct field: %s", t.Field(i).Name)
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
	case "text":
		s := v.String()
		if s == "" {
			return fmt.Errorf("%s is not present in config", fieldName)
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
		if rangeRegex.MatchString(validation) {
			matches := rangeRegex.FindStringSubmatch(validation)
			mins := validateFromField(p, matches[2])
			maxs := validateFromField(p, matches[3])
			if err := validateFromRange(v, mins, maxs, matches[1], matches[4]); err != nil {
				return fmt.Errorf("%s is not in the range of %s: %w", fieldName, validation, err)
			}
		} else if oneofRegex.MatchString(validation) {
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
	switch value.Type().Kind() {
	case reflect.Int:
		if mins != "" {
			mn, err := strconv.Atoi(mins)
			if err != nil {
				return err
			}
			min = float64(mn)
		}
		if maxs != "" {
			mx, err := strconv.Atoi(maxs)
			if err != nil {
				return err
			}
			max = float64(mx)
		}
		val = float64(value.Int())
	case reflect.Float64:
		if mins != "" {
			mn, err := strconv.ParseFloat(mins, 64)
			if err != nil {
				return err
			}
			min = mn
		}
		if maxs != "" {
			mx, err := strconv.Atoi(maxs)
			if err != nil {
				return err
			}
			min = float64(mx)
		}
		val = value.Float()
	default:
		return errors.New("could not validate this value within a range")
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

func validateFromField(value reflect.Value, valuestr string) string {
	if len(valuestr) > 0 && valuestr[0] == '$' {
		v := value.FieldByName(valuestr[1:])
		switch v.Type().Kind() {
		case reflect.Int:
			return fmt.Sprintf("%d", v.Int())
		case reflect.Float64:
			return fmt.Sprintf("%f", v.Float())
		}
	}
	return valuestr
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
	return fmt.Errorf("value is not one of valid values")
}

func isAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}
