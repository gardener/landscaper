// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/errors"
)

// VersionedElement describes an element that has a name and a version.
type VersionedElement interface {
	// GetName gets the name of the element
	GetName() string
	// GetVersion gets the version of the element
	GetVersion() string
}

type NameVersion struct {
	name    string
	version string
}

var _ VersionedElement = (*NameVersion)(nil)

func NewNameVersion(name, version string) NameVersion {
	return NameVersion{name, version}
}

func VersionedElementKey(v VersionedElement) NameVersion {
	if k, ok := v.(NameVersion); ok {
		return k
	}
	return NameVersion{v.GetName(), v.GetVersion()}
}

func (n NameVersion) GetName() string {
	return n.name
}

func (n NameVersion) GetVersion() string {
	return n.version
}

func (n NameVersion) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%s:%s", n.GetName(), n.GetVersion()))
}

func (n NameVersion) Compare(o NameVersion) int {
	c := strings.Compare(n.name, o.name)
	if c == 0 {
		return strings.Compare(n.version, o.version)
	}
	return c
}

func (n NameVersion) String() string {
	if n.version == "" {
		return n.name
	}
	if n.name == "" {
		return n.version
	}
	return n.name + ":" + n.version
}

func ParseNameVersion(s string) (NameVersion, error) {
	a := strings.Split(s, ":")
	if len(a) != 2 {
		return NameVersion{}, errors.ErrInvalid("name:version", s)
	}
	return NewNameVersion(strings.TrimSpace(a[0]), strings.TrimSpace(a[1])), nil
}
