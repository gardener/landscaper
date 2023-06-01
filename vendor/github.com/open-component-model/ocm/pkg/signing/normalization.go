// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"encoding/json"

	"github.com/open-component-model/ocm/pkg/errors"
)

type null struct{}

func (n *null) IsEmpty() bool {
	return true
}

func (n *null) Marshal(gap string) ([]byte, error) {
	return json.Marshal(nil)
}

func (n *null) ToString(gap string) string {
	return n.String()
}

func (n *null) String() string {
	return "null"
}

func (n *null) Formatted() string {
	return n.String()
}

func (n *null) Append(normalized Normalized) {
	panic("append on null")
}

func (n *null) Value() interface{} {
	return nil
}

func (n *null) SetField(name string, value Normalized) {
	panic("set field on null")
}

var Null Normalized = (*null)(nil)

func PrepareNormalization(n Normalization, v interface{}, excludes ExcludeRules) (Normalized, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}

	err = json.Unmarshal(data, &raw)
	if err != nil {
		return nil, err
	}

	return prepareStruct(n, raw, excludes)
}

func Prepare(n Normalization, v interface{}, ex ExcludeRules) (r Normalized, err error) {
	switch e := v.(type) {
	case map[string]interface{}:
		r, err = prepareStruct(n, e, ex)
	case []interface{}:
		r, err = prepareArray(n, e, ex)
	default:
		return n.NewValue(v), nil
	}
	if err != nil {
		return r, err
	}
	if f, ok := ex.(NormalizationFilter); ok {
		return f.Filter(r)
	}
	return r, err
}

func prepareStruct(n Normalization, v map[string]interface{}, ex ExcludeRules) (Normalized, error) {
	entries := n.NewMap()

	for key, value := range v {
		name, mapped, prop := ex.Field(key, value)
		if name != "" {
			nested, err := Prepare(n, mapped, prop)
			if err != nil {
				return nil, errors.Wrapf(err, "field %q", key)
			}
			if nested != nil {
				if nested == Null {
					entries.SetField(name, nil)
				} else {
					entries.SetField(name, nested)
				}
			}
		}
	}
	return entries, nil
}

func prepareArray(n Normalization, v []interface{}, ex ExcludeRules) (Normalized, error) {
	entries := n.NewArray()
	for index, value := range v {
		exclude, mapped, prop := ex.Element(value)
		if !exclude {
			nested, err := Prepare(n, mapped, prop)
			if err != nil {
				return nil, errors.Wrapf(err, "entry %d", index)
			}
			if nested != nil {
				entries.Append(nested)
			}
		}
	}
	return entries, nil
}

func Normalize(n Normalization, v interface{}, ex ExcludeRules) ([]byte, error) {
	entries, err := PrepareNormalization(n, v, ex)
	if err != nil {
		return nil, err
	}
	return entries.Marshal("")
}
