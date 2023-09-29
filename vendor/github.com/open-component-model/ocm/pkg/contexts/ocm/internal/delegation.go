// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sort"
	"sync"

	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

// RepositoryDelegationRegistry is used to register handlers able to dynamically
// enrich the set of available OCM repository types, which are supported without
// explicit registration at the OCM repository type registry. The definition of
// such types is *delegated* to the delegation handler.
// For example, it is used to fade in all the OCI repositories types provided by an
// OCI context and mapped by the genericocireg repository mapping.
// It is used as default decoder for the OCM repository type scheme.
// The encoding is done by the spec objects on their own behalf. Therefore,
// multi version spec types MUST correctly implement the MarshalJSOM method
// on the internal version.
type RepositoryDelegationRegistry = DelegationRegistry[Context, RepositorySpec]

var DefaultRepositoryDelegationRegistry = NewDelegationRegistry[Context, RepositorySpec]()

type PriorityDecoder[C any, T runtime.TypedObject] interface {
	Decode(ctx C, data []byte, unmarshaler runtime.Unmarshaler) (T, error)
	Priority() int
}

type DelegationRegistry[C any, T runtime.TypedObject] interface {
	Register(name string, decoder PriorityDecoder[C, T])
	Get(name string) PriorityDecoder[C, T]
	Delegations() map[string]PriorityDecoder[C, T]
	Decode(ctx C, data []byte, unmarshaller runtime.Unmarshaler) (T, error)

	Copy() DelegationRegistry[C, T]
}

type delegationRegistry[C any, T runtime.TypedObject] struct {
	lock     sync.Mutex
	base     DelegationRegistry[C, T]
	decoders map[string]PriorityDecoder[C, T]
}

var _ DelegationRegistry[Context, RepositorySpec] = (*delegationRegistry[Context, RepositorySpec])(nil)

func NewDelegationRegistry[C any, T runtime.TypedObject](base ...DelegationRegistry[C, T]) DelegationRegistry[C, T] {
	return &delegationRegistry[C, T]{
		decoders: map[string]PriorityDecoder[C, T]{},
		base:     utils.Optional(base...),
	}
}

func (d *delegationRegistry[C, T]) Register(name string, decoder PriorityDecoder[C, T]) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.decoders[name] = decoder
}

func (d *delegationRegistry[C, T]) Get(name string) PriorityDecoder[C, T] {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.decoders[name]
}

func (d *delegationRegistry[C, T]) Copy() DelegationRegistry[C, T] {
	return &delegationRegistry[C, T]{
		base:     d.base,
		decoders: d.Delegations(),
	}
}

func (d *delegationRegistry[C, T]) Delegations() map[string]PriorityDecoder[C, T] {
	d.lock.Lock()
	defer d.lock.Unlock()

	var res map[string]PriorityDecoder[C, T]
	if d.base != nil {
		res = d.base.Delegations()
	} else {
		res = map[string]PriorityDecoder[C, T]{}
	}
	for k, v := range d.decoders {
		res[k] = v
	}
	return res
}

func (d *delegationRegistry[C, T]) Decode(ctx C, data []byte, unmarshaller runtime.Unmarshaler) (T, error) {
	var zero T

	var list []PriorityDecoder[C, T]

	delegates := d.Delegations()
	names := utils.StringMapKeys(delegates)

	for _, n := range names {
		list = append(list, delegates[n])
	}
	if len(list) > 1 {
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].Priority() > list[j].Priority()
		})
	}

	for _, e := range list {
		spec, err := e.Decode(ctx, data, unmarshaller)
		if err != nil {
			return zero, err
		}
		if !runtime.IsUnknown(spec) {
			return spec, nil
		}
	}
	return zero, nil
}
