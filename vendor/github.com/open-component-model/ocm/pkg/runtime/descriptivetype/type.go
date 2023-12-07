// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package descriptivetype

import (
	"fmt"
	"strings"

	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

// DescriptionExtender provides an additional descrition for a type object
// which is appended to the format description in the schmeme descrition
// for the type in question.
type DescriptionExtender[T any] func(t T) string

// TypedObjectType is the appropriately extended type interface
// based on runtime.VersionTypedObjectType providing support for a functional and
// format description.
type TypedObjectType[T runtime.VersionedTypedObject] interface {
	runtime.VersionedTypedObjectType[T]

	Description() string
	Format() string
}

////////////////////////////////////////////////////////////////////////////////

// TypeScheme is the appropriately extended scheme interface based on
// runtime.TypeScheme. Based on the additional type info a complete
// scheme description can be created calling the Describe method.
type TypeScheme[T runtime.VersionedTypedObject, R TypedObjectType[T]] interface {
	runtime.TypeScheme[T, R]

	Describe() string
}

type _typeScheme[T runtime.VersionedTypedObject, R TypedObjectType[T]] interface {
	runtime.TypeScheme[T, R] // for goland to be able to accept extender argument type
}

type typeScheme[T runtime.VersionedTypedObject, R TypedObjectType[T], S TypeScheme[T, R]] struct {
	name     string
	extender DescriptionExtender[R]
	_typeScheme[T, R]
}

func MustNewDefaultTypeScheme[T runtime.VersionedTypedObject, R TypedObjectType[T], S TypeScheme[T, R]](name string, extender DescriptionExtender[R], unknown runtime.Unstructured, acceptUnknown bool, defaultdecoder runtime.TypedObjectDecoder[T], base ...TypeScheme[T, R]) TypeScheme[T, R] {
	scheme := runtime.MustNewDefaultTypeScheme[T, R](unknown, acceptUnknown, defaultdecoder, utils.Optional(base...))
	return &typeScheme[T, R, S]{
		name:        name,
		extender:    extender,
		_typeScheme: scheme,
	}
}

// NewTypeScheme provides an TypeScheme implementation based on the interfaces
// and the default runtime.TypeScheme implementation.
func NewTypeScheme[T runtime.VersionedTypedObject, R TypedObjectType[T], S TypeScheme[T, R]](name string, extender DescriptionExtender[R], unknown runtime.Unstructured, acceptUnknown bool, base ...S) TypeScheme[T, R] {
	scheme := runtime.MustNewDefaultTypeScheme[T, R](unknown, acceptUnknown, nil, utils.Optional(base...))
	return &typeScheme[T, R, S]{
		name:        name,
		extender:    extender,
		_typeScheme: scheme,
	}
}

func (t *typeScheme[T, R, S]) KnownTypes() runtime.KnownTypes[T, R] {
	return t._typeScheme.KnownTypes() // Goland
}

////////////////////////////////////////////////////////////////////////////////

func (t *typeScheme[T, R, S]) Describe() string {
	s := ""
	type method struct {
		desc     string
		versions map[string]string
		more     string
	}

	descs := map[string]*method{}

	// gather info for kinds and versions
	for _, n := range t.KnownTypeNames() {
		kind, vers := runtime.KindVersion(n)

		info := descs[kind]
		if info == nil {
			info = &method{versions: map[string]string{}}
			descs[kind] = info
		}

		if vers == "" {
			vers = "v1"
		}
		if _, ok := info.versions[vers]; !ok {
			info.versions[vers] = ""
		}

		ty := t.GetType(n)

		if t.extender != nil {
			more := t.extender(ty)
			if more != "" {
				info.more = more
			}
		}
		desc := ty.Description()
		if desc != "" {
			info.desc = desc
		}

		desc = ty.Format()
		if desc != "" {
			info.versions[vers] = desc
		}
	}

	for _, tn := range utils.StringMapKeys(descs) {
		info := descs[tn]
		desc := strings.Trim(info.desc, "\n")
		if desc != "" {
			s = fmt.Sprintf("%s\n- %s <code>%s</code>\n\n%s\n\n", s, t.name, tn, utils.IndentLines(desc, "  "))

			format := ""
			for _, f := range utils.StringMapKeys(info.versions) {
				desc = strings.Trim(info.versions[f], "\n")
				if desc != "" {
					format = fmt.Sprintf("%s\n- Version <code>%s</code>\n\n%s\n", format, f, utils.IndentLines(desc, "  "))
				}
			}
			if format != "" {
				s += fmt.Sprintf("  The following versions are supported:\n%s\n", strings.Trim(utils.IndentLines(format, "  "), "\n"))
			}
		}
		s += info.more
	}
	return s
}

////////////////////////////////////////////////////////////////////////////////

type descriptiveTypeInfo interface {
	Description() string
	Format() string
}

type TypedObjectTypeObject[T runtime.VersionedTypedObject] struct {
	runtime.VersionedTypedObjectType[T]
	description string
	format      string
	validator   func(T) error
}

var _ descriptiveTypeInfo = (*TypedObjectTypeObject[runtime.VersionedTypedObject])(nil)

func NewTypedObjectTypeObject[E runtime.VersionedTypedObject](vt runtime.VersionedTypedObjectType[E], opts ...Option) *TypedObjectTypeObject[E] {
	t := NewTypeObjectTarget[E](&TypedObjectTypeObject[E]{
		VersionedTypedObjectType: vt,
	})
	for _, o := range opts {
		o.ApplyTo(t)
	}
	return t.target
}

func (t *TypedObjectTypeObject[T]) Description() string {
	return t.description
}

func (t *TypedObjectTypeObject[T]) Format() string {
	return t.format
}

func (t *TypedObjectTypeObject[T]) Validate(e T) error {
	if t.validator == nil {
		return nil
	}
	return t.validator(e)
}
