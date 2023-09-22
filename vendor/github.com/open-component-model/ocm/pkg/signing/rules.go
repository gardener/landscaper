// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

type Normalization interface {
	NewArray() Normalized
	NewMap() Normalized
	NewValue(v interface{}) Normalized

	String() string
}

type Normalized interface {
	Value() interface{}
	IsEmpty() bool
	Marshal(gap string) ([]byte, error)

	ToString(gap string) string
	String() string
	Formatted() string

	Append(Normalized)
	SetField(name string, value Normalized)
}

// ExcludeRules defines the rules for normalization excludes.
type ExcludeRules interface {
	Field(name string, value interface{}) (string, interface{}, ExcludeRules)
	Element(v interface{}) (bool, interface{}, ExcludeRules)
}

// ValueMappingRule is an optional interface to implement
// to map a value before it is applied to the actual rule.
type ValueMappingRule interface {
	MapValue(v interface{}) interface{}
}

type NormalizationFilter interface {
	Filter(Normalized) (Normalized, error)
}

////////////////////////////////////////////////////////////////////////////////

var (
	_ ExcludeRules     = MapValue{}
	_ ValueMappingRule = MapValue{}
)

type MapValue struct {
	Mapping  ValueMapper
	Continue ExcludeRules
}

func (m MapValue) MapValue(value interface{}) interface{} {
	if m.Mapping != nil {
		return m.Mapping(value)
	}
	return value
}

func (m MapValue) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	if m.Continue != nil {
		return m.Continue.Field(name, value)
	}
	return name, value, NoExcludes{}
}

func (m MapValue) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	if m.Continue != nil {
		return m.Continue.Element(value)
	}
	return true, value, NoExcludes{}
}

////////////////////////////////////////////////////////////////////////////////

type NoExcludes struct{}

var _ ExcludeRules = NoExcludes{}

func (r NoExcludes) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	return name, value, r
}

func (r NoExcludes) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	return false, value, r
}

////////////////////////////////////////////////////////////////////////////////

type ExcludeEmpty struct {
	ExcludeRules
}

var (
	_ ExcludeRules        = ExcludeEmpty{}
	_ NormalizationFilter = ExcludeEmpty{}
)

func (e ExcludeEmpty) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	if e.ExcludeRules == nil {
		if value == nil {
			return "", nil, e
		}
		return name, value, e
	}
	return e.ExcludeRules.Field(name, value)
}

func (e ExcludeEmpty) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	if e.ExcludeRules == nil {
		if value == nil {
			return true, nil, e
		}
		return false, value, e
	}
	return e.ExcludeRules.Element(value)
}

func (ExcludeEmpty) Filter(v Normalized) (Normalized, error) {
	if v == nil {
		return nil, nil
	}
	if v.IsEmpty() {
		return nil, nil
	}
	return v, nil
}

////////////////////////////////////////////////////////////////////////////////

type MapIncludes map[string]ExcludeRules

var _ ExcludeRules = MapIncludes{}

func (r MapIncludes) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	c, ok := r[name]
	if ok {
		if c == nil {
			c = NoExcludes{}
		}
		return name, value, c
	}
	return "", nil, nil
}

func (r MapIncludes) Element(v interface{}) (bool, interface{}, ExcludeRules) {
	panic("invalid exclude structure, require arry but found struct rules")
}

////////////////////////////////////////////////////////////////////////////////

type MapExcludes map[string]ExcludeRules

var _ ExcludeRules = MapExcludes{}

func (r MapExcludes) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	c, ok := r[name]
	if ok {
		if c == nil {
			return "", nil, nil
		}
	} else {
		c = NoExcludes{}
	}
	return name, value, c
}

func (r MapExcludes) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	panic("invalid exclude structure, require array but found struct rules")
}

////////////////////////////////////////////////////////////////////////////////

// DefaultedMapFields can be used to default map fields in maps or lists.
// For map entries the Name field must be set to the fieldname to default.
// For list entries the Name field is ignored.
type DefaultedMapFields struct {
	// Name of the field for map entries, for lists this field must be left blank.
	Name     string
	Fields   map[string]interface{}
	Continue ExcludeRules
	Next     ExcludeRules
}

var _ ExcludeRules = DefaultedMapFields{}

func (r *DefaultedMapFields) setup() {
	if r.Continue == nil && r.Next == nil {
		r.Continue = NoExcludes{}
	}
}

func (r DefaultedMapFields) EnforceNull(fields ...string) DefaultedMapFields {
	if r.Fields == nil {
		r.Fields = map[string]interface{}{}
	}
	for _, f := range fields {
		r.Fields[f] = Null
	}
	return r
}

func (r DefaultedMapFields) EnforceEmptyMap(fields ...string) DefaultedMapFields {
	if r.Fields == nil {
		r.Fields = map[string]interface{}{}
	}
	for _, f := range fields {
		r.Fields[f] = map[string]interface{}{}
	}
	return r
}

func (r DefaultedMapFields) EnforceEmptyList(fields ...string) DefaultedMapFields {
	if r.Fields == nil {
		r.Fields = map[string]interface{}{}
	}
	for _, f := range fields {
		r.Fields[f] = []interface{}{}
	}
	return r
}

func (r DefaultedMapFields) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	if name == r.Name {
		if m, ok := value.(map[string]interface{}); ok {
			for n, v := range r.Fields {
				if m[n] == nil {
					m[n] = v
				}
			}
			value = m
		}
	}
	r.setup()
	if r.Next != nil {
		return r.Next.Field(name, value)
	}
	return name, value, r.Continue
}

func (r DefaultedMapFields) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	if m, ok := value.(map[string]interface{}); ok {
		for n, v := range r.Fields {
			if m[n] == nil {
				m[n] = v
			}
		}
		value = m
	}
	r.setup()
	if r.Next != nil {
		return r.Next.Element(value)
	}
	return false, value, r.Continue
}

////////////////////////////////////////////////////////////////////////////////

type DefaultedListEntries struct {
	Default  interface{}
	Continue ExcludeRules
	Next     ExcludeRules
}

var _ ExcludeRules = DefaultedListEntries{}

func (r *DefaultedListEntries) setup() {
	if r.Continue == nil && r.Next == nil {
		r.Continue = NoExcludes{}
	}
}

func (r DefaultedListEntries) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	panic("invalid exclude structure, require array but found struct rules")
}

func (r DefaultedListEntries) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	if value == nil {
		value = r.Default
	}
	r.setup()
	if r.Next != nil {
		return r.Next.Element(value)
	}
	return false, value, r.Continue
}

////////////////////////////////////////////////////////////////////////////////

type (
	ValueMapper  func(v interface{}) interface{}
	ValueChecker func(value interface{}) bool
)

type DynamicInclude struct {
	ValueChecker ValueChecker
	ValueMapper  ValueMapper
	Continue     ExcludeRules
	Name         string
}

func (r *DynamicInclude) Check(value interface{}) bool {
	return r == nil || r.ValueChecker == nil || r.ValueChecker(value)
}

type DynamicMapIncludes map[string]*DynamicInclude

var _ ExcludeRules = DynamicMapIncludes{}

func (r DynamicMapIncludes) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	e, ok := r[name]
	if ok && e.Check(value) {
		var c ExcludeRules = NoExcludes{}
		if e != nil {
			if e.Name != "" {
				name = e.Name
			}
			if e.Continue != nil {
				c = e.Continue
			}
			if e.ValueMapper != nil {
				value = e.ValueMapper(value)
			}
		}
		return name, value, c
	}
	return "", nil, nil
}

func (r DynamicMapIncludes) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	panic("invalid exclude structure, require arry but found struct rules")
}

////////////////////////////////////////////////////////////////////////////////

type DynamicExclude struct {
	ValueChecker ValueChecker
	ValueMapper  ValueMapper
	Continue     ExcludeRules
	Name         string
}

func (r *DynamicExclude) Check(value interface{}) bool {
	return r == nil || (r.ValueChecker != nil && r.ValueChecker(value)) || (r.ValueChecker == nil && r.Continue == nil)
}

type DynamicMapExcludes map[string]*DynamicExclude

var _ ExcludeRules = DynamicMapExcludes{}

func (r DynamicMapExcludes) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	var c ExcludeRules
	e, ok := r[name]
	if ok {
		if e.Check(value) {
			return "", nil, nil
		}
		if e.Name != "" {
			name = e.Name
		}
		c = e.Continue
	} else {
		c = NoExcludes{}
	}
	if e != nil && e.ValueMapper != nil {
		value = e.ValueMapper(value)
	}
	return name, value, c
}

func (r DynamicMapExcludes) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	panic("invalid exclude structure, require arry but found struct rules")
}

////////////////////////////////////////////////////////////////////////////////

type ConditionalExclude struct {
	ValueChecker  ValueChecker
	ValueMapper   ValueMapper
	ContinueTrue  ExcludeRules
	ContinueFalse ExcludeRules
	Name          string
}

func (r ConditionalExclude) Check(value interface{}) ExcludeRules {
	if r.ValueChecker != nil && r.ValueChecker(value) {
		return r.ContinueTrue
	} else {
		return r.ContinueFalse
	}
}

type ConditionalMapExcludes map[string]*ConditionalExclude

var _ ExcludeRules = ConditionalMapExcludes{}

func (r ConditionalMapExcludes) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	var c ExcludeRules
	e, ok := r[name]
	if ok {
		c = e.Check(value)
		if c == nil {
			return "", nil, nil
		}
		if e.Name != "" {
			name = e.Name
		}
	} else {
		c = NoExcludes{}
	}
	if e != nil && e.ValueMapper != nil {
		value = e.ValueMapper(value)
	}
	return name, value, c
}

func (r ConditionalMapExcludes) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	panic("invalid exclude structure, require arry but found struct rules")
}

////////////////////////////////////////////////////////////////////////////////

type DynamicArrayExcludes struct {
	ValueChecker ValueChecker
	ValueMapper  ValueMapper
	Continue     ExcludeRules
}

var _ ExcludeRules = DynamicArrayExcludes{}

func (r DynamicArrayExcludes) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	panic("invalid exclude structure, require struct but found array rules")
}

func (r DynamicArrayExcludes) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	excl := r.Check(value)
	if !excl && r.ValueMapper != nil {
		value = r.ValueMapper(value)
	}
	if excl || r.Continue != nil {
		return excl, value, r.Continue
	}
	return false, value, NoExcludes{}
}

func (r DynamicArrayExcludes) Check(value interface{}) bool {
	return r.Continue == nil || (r.ValueChecker != nil && r.ValueChecker(value))
}

////////////////////////////////////////////////////////////////////////////////

type ConditionalArrayExcludes struct {
	ValueChecker  ValueChecker
	ValueMapper   ValueMapper
	ContinueTrue  ExcludeRules
	ContinueFalse ExcludeRules
}

var _ ExcludeRules = ConditionalArrayExcludes{}

func (r ConditionalArrayExcludes) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	panic("invalid exclude structure, require struct but found array rules")
}

func (r ConditionalArrayExcludes) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	cont := r.Check(value)
	if cont == nil {
		return true, value, nil
	}
	if r.ValueMapper != nil {
		value = r.ValueMapper(value)
	}
	return false, value, cont
}

func (r ConditionalArrayExcludes) Check(value interface{}) ExcludeRules {
	if r.ValueChecker != nil && r.ValueChecker(value) {
		return r.ContinueTrue
	} else {
		return r.ContinueFalse
	}
}

////////////////////////////////////////////////////////////////////////////////

type ArrayExcludes struct {
	Continue ExcludeRules
}

var _ ExcludeRules = ArrayExcludes{}

func (r ArrayExcludes) Field(name string, value interface{}) (string, interface{}, ExcludeRules) {
	panic("invalid exclude structure, require struct but found array rules")
}

func (r ArrayExcludes) Element(value interface{}) (bool, interface{}, ExcludeRules) {
	return false, value, r.Continue
}
