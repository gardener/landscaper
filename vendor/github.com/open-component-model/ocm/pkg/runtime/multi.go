// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

type multiFormatVersion[T VersionedTypedObject] struct {
	kind    string
	formats FormatVersionRegistry[T]
}

func MustNewMultiFormatVersion[T VersionedTypedObject](kind string, formats FormatVersionRegistry[T]) FormatVersion[T] {
	v, err := NewMultiFormatVersion[T](kind, formats)
	if err != nil {
		panic(err)
	}
	return v
}

// NewMultiFormatVersion is an anonymous version, which covers a set of formats.
// It also accepts legacy formats with a legacy kind.
func NewMultiFormatVersion[T VersionedTypedObject](kind string, formats FormatVersionRegistry[T]) (FormatVersion[T], error) {
	k, v := KindVersion(kind)
	if k != kind {
		return nil, fmt.Errorf("%s is no base type name (should be %s, but has version %s)", kind, k, v)
	}
	// check, whether for all formats the appropriate anonymous format is present
	aliases := utils.StringSet{}
	found := utils.StringSet{}
	for n := range formats.KnownFormats() {
		k, _ := KindVersion(n)
		if n == k {
			found.Add(k)
		}
		aliases.Add(k)
	}
	aliases.Add(kind)
	if !reflect.DeepEqual(aliases, found) {
		for k := range found {
			aliases.Remove(k)
		}
		return nil, errors.Newf("missing base formats %s", utils.StringMapKeys(aliases))
	}
	return &multiFormatVersion[T]{kind, formats}, nil
}

func (m *multiFormatVersion[T]) Encode(t T, marshaler Marshaler) ([]byte, error) {
	return m.formats.Encode(t, marshaler)
}

func (m *multiFormatVersion[T]) Decode(data []byte, unmarshaler Unmarshaler) (T, error) {
	var u UnstructuredVersionedTypedObject

	if unmarshaler == nil {
		unmarshaler = DefaultYAMLEncoding
	}

	var _nil T
	err := unmarshaler.Unmarshal(data, &u)
	if err != nil {
		return _nil, err
	}

	if u.GetType() != u.GetKind() || m.formats.GetFormat(u.GetType()) == nil {
		return _nil, errors.ErrUnknown(errors.KIND_OBJECTTYPE, u.GetType())
	}

	var def T
	found := false
	var defErr error

	for n, t := range m.formats.KnownFormats() {
		k, _ := KindVersion(n)
		if k == u.GetType() {
			e, err := t.Decode(data, unmarshaler)
			if err != nil {
				continue
			}
			if n == u.GetType() {
				found = true
				def = e
				defErr = err
			}
			d, err := t.Encode(e, DefaultJSONEncoding)
			if err != nil {
				continue
			}

			var uc UnstructuredMap
			err = DefaultJSONEncoding.Unmarshal(d, &uc)
			if err != nil {
				continue
			}
			if u.Object.Match(uc) {
				return e, nil
			}
		}
	}
	if found {
		return def, defErr
	}
	return _nil, errors.ErrUnknown(errors.KIND_OBJECTTYPE, u.GetType())
}

////////////////////////////////////////////////////////////////////////////////

type multiFormatVersionType[T VersionedTypedObject, R VersionedTypedObjectType[T]] struct {
	kind string
	FormatVersion[T]
}

func NewMultiFormatVersionedType[T VersionedTypedObject, R VersionedTypedObjectType[T]](kind string, versions TypeVersionScheme[T, R]) (VersionedTypedObjectType[T], error) {
	reg := NewFormatVersionRegistry[T]()
	for n, t := range versions.KnownTypes() {
		reg.Register(n, t)
	}
	vers, err := NewMultiFormatVersion[T](kind, reg)
	if err != nil {
		return nil, err
	}
	return &multiFormatVersionType[T, R]{kind, vers}, nil
}

func (m *multiFormatVersionType[T, R]) GetType() string {
	return m.kind
}

func (m *multiFormatVersionType[T, R]) GetKind() string {
	return m.kind
}

func (m *multiFormatVersionType[T, R]) GetVersion() string {
	return ""
}

////////////////////////////////////////////////////////////////////////////////

type FormatVersionRegistry[I VersionedTypedObject] interface {
	FormatVersion[I]
	Register(string, FormatVersion[I])
	KnownFormats() map[string]FormatVersion[I]
	GetFormat(name string) FormatVersion[I]
}

type formatVersionRegistry[I VersionedTypedObject] struct {
	lock    sync.Mutex
	formats map[string]FormatVersion[I]
}

func NewFormatVersionRegistry[I VersionedTypedObject]() FormatVersionRegistry[I] {
	return &formatVersionRegistry[I]{
		formats: map[string]FormatVersion[I]{},
	}
}

func (r *formatVersionRegistry[I]) Register(name string, fmt FormatVersion[I]) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.formats[name] = fmt
}

func (r *formatVersionRegistry[I]) GetFormat(name string) FormatVersion[I] {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.formats[name]
}

func (r *formatVersionRegistry[I]) Decode(data []byte, unmarshaler Unmarshaler) (I, error) {
	var u UnstructuredTypedObject

	if unmarshaler == nil {
		unmarshaler = DefaultYAMLEncoding
	}

	var _nil I
	err := unmarshaler.Unmarshal(data, &u)
	if err != nil {
		return _nil, err
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if fmt := r.formats[u.GetType()]; fmt != nil {
		return fmt.Decode(data, unmarshaler)
	}
	return _nil, errors.ErrNotSupported(errors.KIND_OBJECTTYPE, u.GetType())
}

func (r *formatVersionRegistry[I]) Encode(obj I, marshaler Marshaler) ([]byte, error) {
	if marshaler == nil {
		marshaler = DefaultYAMLEncoding
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	if fmt := r.formats[obj.GetType()]; fmt != nil {
		return fmt.Encode(obj, marshaler)
	}
	return nil, errors.ErrNotSupported(errors.KIND_OBJECTTYPE, obj.GetType())
}

func (r *formatVersionRegistry[I]) KnownFormats() map[string]FormatVersion[I] {
	r.lock.Lock()
	defer r.lock.Unlock()
	res := map[string]FormatVersion[I]{}

	for k, v := range r.formats {
		res[k] = v
	}
	return res
}
