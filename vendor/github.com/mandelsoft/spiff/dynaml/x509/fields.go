package x509

import (
	"fmt"
	"github.com/mandelsoft/spiff/yaml"
	"strconv"
)

func getField(fields map[string]yaml.Node, name string) interface{} {
	field := fields[name]
	if field == nil {
		return nil
	}
	return field.Value()
}

func getDefaultedBoolField(fields map[string]yaml.Node, name string, def bool) (bool, error) {
	v := getField(fields, name)
	if v == nil {
		return def, nil
	}
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		return strconv.ParseBool(b)
	case int64:
		return b != 0, nil
	default:
		return def, fmt.Errorf("invalid type for boolean field %q", name)
	}
}

func getDefaultedStringField(fields map[string]yaml.Node, name string, def string) (string, error) {
	v := getField(fields, name)
	if v == nil {
		return def, nil
	}
	switch f := v.(type) {
	case string:
		return f, nil
	case bool:
		return strconv.FormatBool(f), nil
	case int64:
		return strconv.FormatInt(f, 10), nil
	default:
		return "", fmt.Errorf("invalid type for %q", name)
	}
}

func getDefaultedIntField(fields map[string]yaml.Node, name string, def int64) (int64, error) {
	v := getField(fields, name)
	if v == nil {
		return def, nil
	}
	switch f := v.(type) {
	case string:
		return strconv.ParseInt(f, 10, 64)
	case int64:
		return f, nil
	default:
		return 0, fmt.Errorf("invalid type for %q", name)
	}
}

func getDefaultedStringListField(fields map[string]yaml.Node, name string, def []string) ([]string, error) {
	v := getField(fields, name)
	if v == nil {
		return def, nil
	}
	switch f := v.(type) {
	case string:
		return []string{f}, nil
	case bool:
		return []string{strconv.FormatBool(f)}, nil
	case int64:
		return []string{strconv.FormatInt(f, 10)}, nil
	case []yaml.Node:
		r := make([]string, len(f))
		for i, e := range f {
			switch ev := e.Value().(type) {
			case string:
				r[i] = ev
			case bool:
				r[i] = strconv.FormatBool(ev)
			case int64:
				r[i] = strconv.FormatInt(ev, 10)
			default:
				return nil, fmt.Errorf("invalid list element type for %q", name)
			}
		}
		return r, nil
	default:
		return nil, fmt.Errorf("invalid type for %q", name)
	}
}

func getStringListField(fields map[string]yaml.Node, name string) ([]string, error) {
	l, err := getDefaultedStringListField(fields, name, nil)
	if l == nil && err != nil {
		return nil, fmt.Errorf("field %q is required", name)
	}
	return l, err
}
