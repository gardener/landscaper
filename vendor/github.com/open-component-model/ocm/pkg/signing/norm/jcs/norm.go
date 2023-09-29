// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package jcs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/signing"
	"github.com/open-component-model/ocm/pkg/utils"
)

var Type = normalization{}

type normalization struct{}

func New() signing.Normalization {
	return normalization{}
}

func (_ normalization) NewArray() signing.Normalized {
	return &normalized{[]interface{}{}}
}

func (_ normalization) NewMap() signing.Normalized {
	return &normalized{map[string]interface{}{}}
}

func (_ normalization) NewValue(v interface{}) signing.Normalized {
	return &normalized{v}
}

func (_ normalization) String() string {
	return "JCS(rfc8785) normalization"
}

type normalized struct {
	value interface{}
}

func (n *normalized) Value() interface{} {
	return n.value
}

func (n *normalized) IsEmpty() bool {
	switch v := n.value.(type) {
	case map[string]interface{}:
		return len(v) == 0
	case []interface{}:
		return len(v) == 0
	default:
		return false
	}
}

func (n *normalized) Append(normalized signing.Normalized) {
	n.value = append(n.value.([]interface{}), normalized.Value())
}

func (n *normalized) SetField(name string, value signing.Normalized) {
	v := n.value.(map[string]interface{})
	v[name] = value.Value()
}

func (n *normalized) ToString(gap string) string {
	return toString(n.value, gap)
}

func (l *normalized) String() string {
	return string(utils.Must(json.Marshal(l.value)))
}

func (l *normalized) Formatted() string {
	return string(utils.Must(json.MarshalIndent(l.value, "", "  ")))
}

func (n *normalized) Marshal(gap string) ([]byte, error) {
	byteBuffer := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(byteBuffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", gap)

	err := encoder.Encode(n.Value())
	if err != nil {
		return nil, err
	}
	if gap != "" {
		return byteBuffer.Bytes(), nil
	}
	data, err := jsoncanonicalizer.Transform(byteBuffer.Bytes())
	if err != nil {
		return nil, errors.Wrapf(err, "cannot canonicalize json")
	}
	return data, nil
}

func toString(v interface{}, gap string) string {
	if v == nil || v == signing.Null {
		return "null"
	}
	switch castIn := v.(type) {
	case map[string]interface{}:
		ngap := gap + "  "
		s := "{"
		sep := ""
		keys := utils.StringMapKeys(castIn)
		for _, n := range keys {
			v := castIn[n] //nolint: govet // yes
			sep = "\n" + gap
			s = fmt.Sprintf("%s%s  %s: %s", s, sep, n, toString(v, ngap))
		}
		s += sep + "}"
		return s
	case []interface{}:
		ngap := gap + "  "
		s := "["
		sep := ""
		for _, v := range castIn {
			s = fmt.Sprintf("%s\n%s%s", s, ngap, toString(v, ngap))
			sep = "\n" + gap
		}
		s += sep + "]"
		return s
	case string:
		return castIn
	case bool:
		return strconv.FormatBool(castIn)
	default:
		panic(fmt.Sprintf("unknown type %T in toString. This should not happen", v))
	}
}
