// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/open-component-model/ocm/pkg/contexts/ocm/compdesc/equivalent"
	"github.com/open-component-model/ocm/pkg/logging"
)

// Identity describes the identity of an object.
// Only ascii characters are allowed
// +k8s:deepcopy-gen=true
// +k8s:openapi-gen=true
type Identity map[string]string

func NewExtraIdentity(extras ...string) Identity {
	if len(extras) == 0 {
		return nil
	}
	id := Identity{}
	i := 0
	for i < len(extras) {
		if i+1 < len(extras) {
			id[extras[i]] = extras[i+1]
		} else {
			id[extras[i]] = ""
		}
		i += 2
	}
	return id
}

// NewIdentity return a simple name identity.
func NewIdentity(name string, extras ...string) Identity {
	id := NewExtraIdentity(extras...)
	if id == nil {
		return Identity{SystemIdentityName: name}
	}
	id[SystemIdentityName] = name
	return id
}

// Digest returns the object digest of an identity.
func (i Identity) Digest() []byte {
	data, err := json.Marshal(i)
	if err != nil {
		logging.Logger().LogError(err, "corrupted digest")
	}

	return data
}

// Equals compares two identities.
func (i Identity) Equals(o Identity) bool {
	if len(i) != len(o) {
		return false
	}

	for k, v := range i {
		if v2, ok := o[k]; !ok || v != v2 {
			return false
		}
	}
	return true
}

func (i Identity) Equivalent(o Identity) equivalent.EqualState {
	if len(i) != len(o) {
		return equivalent.StateNotLocalHashEqual()
	}

	for k, v := range i {
		if v2, ok := o[k]; !ok || v != v2 {
			return equivalent.StateNotLocalHashEqual()
		}
	}
	return equivalent.StateEquivalent()
}

func (i *Identity) Set(name, value string) {
	if *i == nil {
		*i = Identity{name: value}
	} else {
		(*i)[name] = value
	}
}

func (i Identity) Get(name string) string {
	if i != nil {
		return i[name]
	}
	return ""
}

func (i Identity) Remove(name string) bool {
	if i != nil {
		delete(i, name)
	}
	return false
}

func (i Identity) String() string {
	if i == nil {
		return ""
	}

	var keys []string
	for k := range i {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	s := ""
	sep := ""
	for _, k := range keys {
		s = fmt.Sprintf("%s%s%q=%q", s, sep, k, i[k])
		sep = ","
	}
	return s
}

// Match implements the selector interface.
func (i Identity) Match(obj map[string]string) (bool, error) {
	for k, v := range i {
		if obj[k] != v {
			return false, nil
		}
	}
	return true, nil
}

// Copy copies identity.
func (i Identity) Copy() Identity {
	if i == nil {
		return nil
	}
	n := Identity{}
	for k, v := range i {
		n[k] = v
	}
	return n
}

// ValidateIdentity validates the identity of object.
func ValidateIdentity(fldPath *field.Path, id Identity) field.ErrorList {
	allErrs := field.ErrorList{}

	for key := range id {
		if key == SystemIdentityName {
			allErrs = append(allErrs, field.Forbidden(fldPath.Key(SystemIdentityName), "name is a reserved system identity label"))
		}

		if !IsASCII(key) {
			allErrs = append(allErrs, field.Forbidden(fldPath.Key(key), "key contains non-ascii characters"))
		}
		if !IsIdentity(key) {
			allErrs = append(allErrs, field.Invalid(fldPath.Key(key), key, IdentityKeyValidationErrMsg))
		}
	}
	return allErrs
}
