// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

type Normalized interface{}

// ExcludeRules defines the rules for normalization excludes.
type ExcludeRules interface {
	Field(name string, value interface{}) (string, interface{}, ExcludeRules)
	Element(v interface{}) (bool, interface{}, ExcludeRules)
}

type NormalizationFilter interface {
	Filter(Normalized) (Normalized, error)
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
	switch r := v.(type) {
	case []Normalized:
		if len(r) == 0 {
			return nil, nil
		}
	case []Entry:
		if len(r) == 0 {
			return nil, nil
		}
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
	panic("invalid exclude structure, require arry but found struct rules")
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

////////////////////////////////////////////////////////////////////////////////

func IgnoreLabelsWithoutSignature(v interface{}) bool {
	if m, ok := v.(map[string]interface{}); ok {
		if sig, ok := m["signing"]; ok {
			if sig != nil {
				return sig != "true" && sig != true
			}
		}
	}
	return true
}

////////////////////////////////////////////////////////////////////////////////

func IgnoreResourcesWithNoneAccess(v interface{}) bool {
	return CheckIgnoreResourcesWithAccessType(func(k string) bool { return k == "none" || k == "None" }, v)
}

func IgnoreResourcesWithAccessType(t string) func(v interface{}) bool {
	return func(v interface{}) bool {
		return CheckIgnoreResourcesWithAccessType(func(k string) bool { return k == t }, v)
	}
}

func CheckIgnoreResourcesWithAccessType(t func(string) bool, v interface{}) bool {
	access := v.(map[string]interface{})["access"]
	if access == nil {
		return true
	}
	typ := access.(map[string]interface{})["type"]
	if s, ok := typ.(string); ok {
		return t(s)
	}
	return false
}
