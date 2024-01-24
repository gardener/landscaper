// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"strings"
)

// TypeGetter is the interface to be implemented for extracting a type.
type TypeGetter interface {
	// GetType returns the type of the access object.
	GetType() string
}

// TypeSetter is the interface to be implemented for extracting a type.
type TypeSetter interface {
	// SetType sets the type of abstract element
	SetType(typ string)
}

type (
	// TypeInfo defines the accessors for type information.
	TypeInfo interface {
		TypeGetter
	}
)

type (
	// ObjectType is the data type providing serializable type information.
	// It is an implementation of TypeInfo.
	ObjectType struct {
		// Type describes the type of the object.
		Type string `json:"type"`
	}
	_ObjectType = ObjectType // provide the same type to be used as provide embedded field
)

var _ TypeInfo = (*ObjectType)(nil)

// NewObjectType creates an ObjectType value.
func NewObjectType(typ string) ObjectType {
	return ObjectType{typ}
}

// GetType returns the type of the object.
func (t ObjectType) GetType() string {
	return t.Type
}

// SetType sets the type of the object.
func (t *ObjectType) SetType(typ string) {
	t.Type = typ
}

////////////////////////////////////////////////////////////////////////////////

type (
	VersionedObjectType  ObjectType
	_VersionedObjectType = VersionedObjectType
)

func NewVersionedObjectType(args ...string) VersionedObjectType {
	return NewVersionedTypedObject(args...)
}

// GetType returns the type of the object.
func (t VersionedObjectType) GetType() string {
	return t.Type
}

// SetType sets the type of the object.
func (t *VersionedObjectType) SetType(typ string) {
	t.Type = typ
}

// GetKind returns the kind of the object.
func (v VersionedObjectType) GetKind() string {
	return GetKind(v)
}

// SetKind sets the kind of the object.
func (v *VersionedObjectType) SetKind(kind string) {
	t := v.GetType()
	i := strings.LastIndex(t, VersionSeparator)
	if i < 0 {
		v.SetType(kind)
	} else {
		v.SetType(kind + t[i:])
	}
}

// GetVersion returns the version of the object.
func (v VersionedObjectType) GetVersion() string {
	return GetVersion(v)
}

// SetVersion sets the version of the object.
func (v *VersionedObjectType) SetVersion(version string) {
	t := v.GetType()
	i := strings.LastIndex(t, VersionSeparator)
	if i < 0 {
		if version != "" {
			v.SetType(v.Type + VersionSeparator + version)
		}
	} else {
		if version != "" {
			v.SetType(t[:i] + VersionSeparator + version)
		} else {
			v.SetType(t[:i])
		}
	}
}
