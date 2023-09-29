// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"encoding/json"
	"fmt"

	"github.com/open-component-model/ocm/pkg/generics"
)

// Properties describes a set of name/value pairs.
type Properties map[string]string

// Digest returns the object digest of a Property set.
func (p Properties) Digest() ([]byte, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to produce digest: %w", err)
	}
	return data, nil
}

func (p Properties) SetNonEmptyValue(name, value string) {
	if value != "" {
		p[name] = value
	}
}

// Equals compares two identities.
func (p Properties) Equals(o Properties) bool {
	if len(p) != len(o) {
		return false
	}

	for k, v := range p {
		if v2, ok := o[k]; !ok || v != v2 {
			return false
		}
	}
	return true
}

// Match implements the selector interface.
func (p Properties) Match(obj map[string]string) (bool, error) {
	for k, v := range p {
		if obj[k] != v {
			return false, nil
		}
	}
	return true, nil
}

// Names returns the set of property names.
func (c Properties) Names() generics.Set[string] {
	return generics.KeySet(c)
}

// String returns a string representation.
func (c Properties) String() string {
	if c == nil {
		return "<none>"
	}
	//nolint: errchkjson // just a string map
	d, _ := json.Marshal(c)
	return string(d)
}

// Copy copies identity.
func (p Properties) Copy() Properties {
	if p == nil {
		return nil
	}
	n := Properties{}
	for k, v := range p {
		n[k] = v
	}
	return n
}
