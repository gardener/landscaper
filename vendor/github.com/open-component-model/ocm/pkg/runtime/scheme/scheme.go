// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package scheme

import (
	"sync"

	"github.com/Masterminds/semver/v3"

	"github.com/open-component-model/ocm/pkg/generics"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

type Object = runtime.VersionedTypedObject

type Kind interface {
	Kind() string
	Description() string
}

// ObjectDecoder is able to provide an effective typed object for some
// serilaized form. The technical deserialization is done by an Unmarshaler.
type ObjectDecoder[O Object] interface {
	Decode(data []byte, unmarshaler runtime.Unmarshaler) (O, error)
}

// ObjectEncoder is able to provide a versioned representation of
// an effective Object.
type ObjectEncoder[O Object] interface {
	Encode(O, runtime.Marshaler) ([]byte, error)
}

type KnownTypes[O Object, T Type[O]] map[string]T

type Type[O Object] interface {
	ObjectDecoder[O]
	ObjectEncoder[O]
}

type Scheme[O Object, T Type[O]] interface {
	RegisterKind(kind Kind)
	RegisterType(name string, otype T) error

	GetType(name string) T
	GetKind(kind string) Kind

	Decode(data []byte, unmarshaler runtime.Unmarshaler) (O, error)
	Encode(obj O, marshaler runtime.Marshaler) ([]byte, error)

	KnownTypes() KnownTypes[O, T]
	KnownKinds() []string
	KnownVersions(kind string) []string
}

type kindInfo struct {
	defined  bool
	kind     Kind
	versions generics.Set[string]
}

type kind struct {
	kind        string
	description string
}

func NewKind(name, desc string) Kind {
	return &kind{
		kind:        name,
		description: desc,
	}
}

func (k *kind) Kind() string {
	return k.kind
}

func (k *kind) Description() string {
	return k.description
}

type scheme[O Object, T Type[O]] struct {
	lock   sync.Mutex
	scheme runtime.Scheme
	kinds  map[string]*kindInfo
}

func NewScheme[O Object, T Type[O]]() Scheme[O, T] {
	var obj O
	s := runtime.MustNewDefaultScheme(&obj, nil, false, nil)
	return &scheme[O, T]{scheme: s, kinds: map[string]*kindInfo{}}
}

func (t *scheme[O, T]) GetType(name string) T {
	var zero T
	d := t.scheme.GetDecoder(name)
	if d == nil {
		return zero
	}
	return d.(runtimeEncodingAdapter[O, T]).typ
}

func (t *scheme[O, T]) GetKind(kind string) Kind {
	t.lock.Lock()
	defer t.lock.Unlock()

	d := t.kinds[kind]
	if d == nil {
		return nil
	}
	return d.kind
}

type runtimeEncodingAdapter[O Object, T Type[O]] struct {
	typ T
}

var (
	_ runtime.TypedObjectDecoder = (*runtimeEncodingAdapter[Object, Type[Object]])(nil)
	_ runtime.TypedObjectEncoder = (*runtimeEncodingAdapter[Object, Type[Object]])(nil)
)

func (r runtimeEncodingAdapter[O, T]) Encode(object runtime.TypedObject, marshaler runtime.Marshaler) ([]byte, error) {
	return r.typ.Encode(object.(O), marshaler)
}

func (r runtimeEncodingAdapter[O, T]) Decode(data []byte, unmarshaler runtime.Unmarshaler) (runtime.TypedObject, error) {
	o, err := r.typ.Decode(data, unmarshaler)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (t *scheme[O, T]) RegisterKind(k Kind) {
	t.lock.Lock()
	defer t.lock.Unlock()

	ki := t.kinds[k.Kind()]
	if ki == nil {
		ki = &kindInfo{
			kind:     k,
			versions: generics.Set[string]{},
		}
	}
	ki.defined = true
}

func (t *scheme[O, T]) RegisterType(name string, typ T) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	k, v := runtime.KindVersion(name)
	wrap := runtimeEncodingAdapter[O, T]{typ}

	if v != "" {
		_, err := semver.NewVersion(v)
		if err != nil {
			return err
		}
	}
	ki := t.kinds[k]
	if ki == nil {
		ki = &kindInfo{
			defined:  false,
			kind:     NewKind(k, "implicitly defined"),
			versions: generics.Set[string]{},
		}
		t.kinds[k] = ki
	}

	if v == "v1" && t.scheme.GetDecoder(k) == nil {
		t.scheme.RegisterByDecoder(k, wrap)
	}
	if v == "" {
		t.scheme.RegisterByDecoder(runtime.TypeName(k, "v1"), wrap)
		ki.versions.Add("v1")
	} else {
		ki.versions.Add(v)
	}
	return t.scheme.RegisterByDecoder(name, wrap)
}

func (t *scheme[O, T]) Decode(data []byte, unmarshaler runtime.Unmarshaler) (O, error) {
	var zero O
	o, err := t.scheme.Decode(data, unmarshaler)
	if err != nil {
		return zero, err
	}
	return o.(O), nil
}

func (t *scheme[O, T]) Encode(obj O, marshaler runtime.Marshaler) ([]byte, error) {
	return t.scheme.Encode(obj, marshaler)
}

func (t *scheme[O, T]) KnownVersions(kind string) []string {
	t.lock.Lock()
	defer t.lock.Unlock()

	ki := t.kinds[kind]
	if ki == nil {
		return nil
	}
	result := ki.versions.AsArray()
	SortVersions(result)
	return result
}

func (t *scheme[O, T]) KnownTypes() KnownTypes[O, T] {
	t.lock.Lock()
	defer t.lock.Unlock()

	result := KnownTypes[O, T]{}

	for k, v := range t.scheme.KnownTypes() {
		result[k] = v.(runtimeEncodingAdapter[O, T]).typ
	}
	return result
}

func (t *scheme[O, T]) KnownKinds() []string {
	t.lock.Lock()
	defer t.lock.Unlock()

	return utils.StringMapKeys(t.kinds)
}
