// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package entry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

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
	return &normalized{Entries{}}
}

func (_ normalization) NewValue(v interface{}) signing.Normalized {
	return &normalized{v}
}

func (_ normalization) String() string {
	return "entry normalization"
}

type normalized struct {
	value interface{}
}

func (n *normalized) Value() interface{} {
	return n.value
}

func (n *normalized) IsEmpty() bool {
	switch v := n.value.(type) {
	case Entries:
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
	v := n.value.(Entries)
	v = append(v, Entry{
		key:   name,
		value: value.Value(),
	})
	// sort the entries based on the key
	sort.SliceStable(v, func(i, j int) bool {
		return v[i].Key() < v[j].Key()
	})
	n.value = v
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

	if err := encoder.Encode(n.value); err != nil {
		return nil, err
	}

	normalizedJson := byteBuffer.Bytes()

	// encoder.Encode appends a newline that we do not want
	if normalizedJson[len(normalizedJson)-1] == 10 {
		normalizedJson = normalizedJson[:len(normalizedJson)-1]
	}
	return normalizedJson, nil
}

// Entry is used to keep exactly one key/value pair.
type Entry struct {
	key   string
	value interface{}
}

func (e Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		e.key: e.value,
	})
}

func NewEntry(key string, value interface{}) Entry {
	return Entry{key: key, value: value}
}

func (e Entry) Get() (string, interface{}) {
	return e.key, e.value
}

func (e Entry) Key() string {
	return e.key
}

func (e Entry) Value() interface{} {
	return e.value
}

func (e Entry) ToString(gap string) string {
	return fmt.Sprintf("%s%s: %s", gap, e.Key(), toString(e.Value(), gap))
}

type Entries []Entry

func (l *Entries) Add(key string, value interface{}) {
	*l = append(*l, NewEntry(key, value))
}

func (l Entries) String() string {
	return string(utils.Must(json.Marshal(l)))
}

func (l Entries) Formatted() string {
	return string(utils.Must(json.MarshalIndent(l, "", "  ")))
}

func (l Entries) ToString(gap string) string {
	ngap := gap + "  "
	s := "{"
	sep := ""
	for _, v := range l {
		s = fmt.Sprintf("%s\n%s", s, v.ToString(ngap))
		sep = "\n" + gap
	}
	s += sep + "}"
	return s
}

func toString(v interface{}, gap string) string {
	if v == nil || v == signing.Null {
		return "null"
	}
	switch castIn := v.(type) {
	case Entries:
		return castIn.ToString(gap)
	case []Entry:
		return Entries(castIn).ToString(gap)
	case Entry:
		return castIn.ToString(gap)
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
