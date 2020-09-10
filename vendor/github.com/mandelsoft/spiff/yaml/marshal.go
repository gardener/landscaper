package yaml

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/mandelsoft/spiff/legacy/candiedyaml"
)

func Marshal(node Node) ([]byte, error) {
	return candiedyaml.Marshal(node)
}

func ToJSON(root Node) ([]byte, error) {
	if root == nil {
		return ValueToJSON(nil)
	}
	return ValueToJSON(root.Value())
}

func ValueToJSON(root interface{}) ([]byte, error) {
	n, err := normalizeValue(root)
	if err != nil {
		return nil, err
	}
	return json.Marshal(n)
}

func Normalize(root Node) (interface{}, error) {
	if root == nil || root.Value() == nil {
		return nil, nil
	}
	return normalizeValue(root.Value())
}

func normalizeValue(value interface{}) (interface{}, error) {
	switch rootVal := value.(type) {
	case candiedyaml.Marshaler:
		_, v, err := rootVal.MarshalYAML()
		if err != nil {
			return nil, err
		}
		return normalizeValue(v)

	case map[string]Node:
		normalized := map[string]interface{}{}

		for key, val := range rootVal {
			sub, err := Normalize(val)
			if err != nil {
				return nil, err
			}

			normalized[key] = sub
		}

		return normalized, nil

	case []Node:
		normalized := []interface{}{}

		for _, val := range rootVal {
			sub, err := Normalize(val)
			if err != nil {
				return nil, err
			}

			normalized = append(normalized, sub)
		}

		return normalized, nil

	case string, []byte, int64, float64, bool, nil:
		return rootVal, nil
	}

	return nil, errors.New(fmt.Sprintf("unknown type (%s) during normalization: %#v\n", reflect.TypeOf(value), value))
}
