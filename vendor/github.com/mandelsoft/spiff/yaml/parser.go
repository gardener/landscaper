package yaml

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/mandelsoft/spiff/legacy/candiedyaml"
)

//	"github.com/cloudfoundry-incubator/candiedyaml"

type NonStringKeyError struct {
	Key interface{}
}

func (e NonStringKeyError) Error() string {
	return fmt.Sprintf("map key must be a string: %#v", e.Key)
}

func Unmarshal(sourceName string, source []byte) (Node, error) {
	return Parse(sourceName, source)
}

func Parse(sourceName string, source []byte) (Node, error) {
	docs, err := ParseMulti(sourceName, source)
	if err != nil {
		return nil, err
	}
	if len(docs) > 1 {
		return nil, fmt.Errorf("multi document not possible")
	}
	return docs[0], err
}

func UnmarshalMulti(sourceName string, source []byte) ([]Node, error) {
	return ParseMulti(sourceName, source)
}

func ParseMulti(sourceName string, source []byte) ([]Node, error) {
	docs := []Node{}

	if len(bytes.Trim(source, " \t\n\r")) == 0 {
		source = []byte("---\n")
	}
	r := bytes.NewBuffer(source)
	d := candiedyaml.NewDecoder(r)

	for d.HasNext() {
		var parsed interface{}
		err := d.Decode(&parsed)
		if err != nil {
			return nil, err
		}
		n, err := Sanitize(sourceName, parsed)
		if err != nil {
			return nil, err
		}
		docs = append(docs, n)
	}
	return docs, nil
}

var mapType = reflect.TypeOf(map[string]interface{}{})
var arrayType = reflect.TypeOf([]interface{}{})

func Sanitize(sourceName string, root interface{}) (Node, error) {
	switch rootVal := root.(type) {
	case time.Time:
		return NewNode(rootVal.Format("2019-01-08T10:06:26Z"), sourceName), nil
	case map[interface{}]interface{}:
		sanitized := map[string]Node{}

		for key, val := range rootVal {
			str, ok := key.(string)
			if !ok {
				return nil, NonStringKeyError{key}
			}

			sub, err := Sanitize(sourceName, val)
			if err != nil {
				return nil, err
			}

			sanitized[str] = sub
		}

		return NewNode(sanitized, sourceName), nil

	case []interface{}:
		sanitized := []Node{}

		for _, val := range rootVal {
			sub, err := Sanitize(sourceName, val)
			if err != nil {
				return nil, err
			}

			sanitized = append(sanitized, sub)
		}

		return NewNode(sanitized, sourceName), nil

	case map[string]interface{}:
		sanitized := map[string]Node{}

		for key, val := range rootVal {
			sub, err := Sanitize(sourceName, val)
			if err != nil {
				return nil, err
			}

			sanitized[key] = sub
		}

		return NewNode(sanitized, sourceName), nil
	case int:
		return NewNode(int64(rootVal), sourceName), nil
	case int32:
		return NewNode(int64(rootVal), sourceName), nil
	case float32:
		return NewNode(float64(rootVal), sourceName), nil
	case string, []byte, int64, float64, bool, nil:
		return NewNode(rootVal, sourceName), nil
	}

	value := reflect.ValueOf(root)
	if value.Type().ConvertibleTo(mapType) {
		return Sanitize(sourceName, value.Convert(mapType).Interface())
	}
	if value.Type().ConvertibleTo(arrayType) {
		return Sanitize(sourceName, value.Convert(arrayType).Interface())
	}
	return nil, errors.New(fmt.Sprintf("unknown type (%s) during sanitization: %#v\n", reflect.TypeOf(root).String(), root))
}
