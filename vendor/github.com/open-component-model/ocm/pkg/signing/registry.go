// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"sort"
	"sync"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"github.com/open-component-model/ocm/pkg/generics"
)

type Registry interface {
	HandlerRegistryFuncs
	KeyRegistryFuncs

	HandlerRegistryProvider
	KeyRegistryProvider

	Copy() Registry
}

type HasherProvider interface {
	GetHasher(name string) Hasher
}

type HasherRegistryFuncs interface {
	HasherProvider

	RegisterHasher(hasher Hasher)
	HasherNames() []string
}

type HasherRegistry interface {
	HasherRegistryFuncs

	Copy() HasherRegistry
}

type HasherRegistryProvider interface {
	HasherRegistry() HasherRegistry
}

type SignerRegistryFuncs interface {
	RegisterSignatureHandler(handler SignatureHandler)
	RegisterSigner(algo string, signer Signer)
	RegisterVerifier(algo string, verifier Verifier)
	GetSigner(name string) Signer
	GetVerifier(name string) Verifier
	SignerNames() []string
}

type SignerRegistry interface {
	SignerRegistryFuncs

	Copy() SignerRegistry
}

type SignerRegistryProvider interface {
	SignerRegistry() SignerRegistry
}

type HandlerRegistryFuncs interface {
	SignerRegistryFuncs
	HasherRegistryFuncs

	SignerRegistryProvider
	HasherRegistryProvider
}

type HandlerRegistry interface {
	HandlerRegistryFuncs
	Copy() HandlerRegistry
}

type KeyRegistryFuncs interface {
	RegisterPublicKey(name string, key interface{})
	RegisterPrivateKey(name string, key interface{})
	GetPublicKey(name string) interface{}
	GetPrivateKey(name string) interface{}

	HasKeys() bool
}

type HandlerRegistryProvider interface {
	HandlerRegistry() HandlerRegistry
}

type KeyRegistry interface {
	KeyRegistryFuncs
	Copy() KeyRegistry
}

type KeyRegistryProvider interface {
	KeyRegistry() KeyRegistry
}

////////////////////////////////////////////////////////////////////////////////

type (
	_hasherRegistry = HasherRegistry
	_signerRegistry = SignerRegistry
)

type handlerRegistry struct {
	_hasherRegistry
	_signerRegistry
}

var _ HandlerRegistry = (*handlerRegistry)(nil)

func NewHandlerRegistry(parents ...HandlerRegistry) HandlerRegistry {
	return &handlerRegistry{
		_hasherRegistry: NewHasherRegistry(generics.ConvertSliceWith(toHasherRegistry, parents)...),
		_signerRegistry: NewSignerRegistry(generics.ConvertSliceWith(toSignerRegistry, parents)...),
	}
}

func toHasherRegistry(o HasherRegistryProvider) HasherRegistry {
	if o == nil {
		return nil
	}
	return o.HasherRegistry()
}

func toSignerRegistry(o SignerRegistryProvider) SignerRegistry {
	if o == nil {
		return nil
	}
	return o.SignerRegistry()
}

func (r *handlerRegistry) Copy() HandlerRegistry {
	return &handlerRegistry{
		_hasherRegistry: r._hasherRegistry.Copy(),
		_signerRegistry: r._signerRegistry.Copy(),
	}
}

func (r *handlerRegistry) HasherRegistry() HasherRegistry {
	return r._hasherRegistry
}

func (r *handlerRegistry) SignerRegistry() SignerRegistry {
	return r._signerRegistry
}

////////////////////////////////////////////////////////////////////////////////

type signerRegistry struct {
	lock      sync.RWMutex
	parents   []SignerRegistry
	signers   map[string]Signer
	verifiers map[string]Verifier
}

var _ SignerRegistry = (*signerRegistry)(nil)

func NewSignerRegistry(parents ...SignerRegistry) SignerRegistry {
	return &signerRegistry{
		parents:   slices.Clone(parents),
		signers:   map[string]Signer{},
		verifiers: map[string]Verifier{},
	}
}

func (r *signerRegistry) Copy() SignerRegistry {
	return &signerRegistry{
		parents:   r.parents,
		signers:   maps.Clone(r.signers),
		verifiers: maps.Clone(r.verifiers),
	}
}

func (r *signerRegistry) RegisterSignatureHandler(handler SignatureHandler) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.signers[handler.Algorithm()] = handler
	r.verifiers[handler.Algorithm()] = handler
}

func (r *signerRegistry) RegisterSigner(algo string, signer Signer) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.signers[algo] = signer
	if v, ok := signer.(Verifier); ok && r.verifiers[algo] == nil {
		r.verifiers[algo] = v
	}
}

func (r *signerRegistry) SignerNames() []string {
	r.lock.Lock()
	defer r.lock.Unlock()

	names := generics.Set[string]{}
	for n := range r.signers {
		names.Add(n)
	}
	for _, p := range r.parents {
		if p == nil {
			continue
		}
		names.Add(p.SignerNames()...)
	}
	result := names.AsArray()
	sort.Strings(result)
	return result
}

func (r *signerRegistry) RegisterVerifier(algo string, verifier Verifier) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.verifiers[algo] = verifier
	if v, ok := verifier.(Signer); ok && r.signers[algo] == nil {
		r.signers[algo] = v
	}
}

func (r *signerRegistry) GetSigner(name string) Signer {
	r.lock.RLock()
	defer r.lock.RUnlock()

	s := r.signers[name]
	if s != nil {
		return s
	}
	for _, p := range r.parents {
		if p == nil {
			continue
		}
		s = p.GetSigner(name)
		if s != nil {
			return s
		}
	}
	return nil
}

func (r *signerRegistry) GetVerifier(name string) Verifier {
	r.lock.RLock()
	defer r.lock.RUnlock()

	v := r.verifiers[name]
	if v != nil {
		return v
	}
	for _, p := range r.parents {
		if p == nil {
			continue
		}
		v = p.GetVerifier(name)
		if v != nil {
			return v
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type hasherRegistry struct {
	lock    sync.RWMutex
	parents []HasherRegistry
	hasher  map[string]Hasher
}

var _ HasherRegistry = (*hasherRegistry)(nil)

func NewHasherRegistry(parents ...HasherRegistry) HasherRegistry {
	return &hasherRegistry{
		parents: slices.Clone(parents),
		hasher:  map[string]Hasher{},
	}
}

func (r *hasherRegistry) Copy() HasherRegistry {
	return &hasherRegistry{
		parents: r.parents,
		hasher:  maps.Clone(r.hasher),
	}
}

func (r *hasherRegistry) RegisterHasher(hasher Hasher) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.hasher[hasher.Algorithm()] = hasher
}

func (r *hasherRegistry) HasherNames() []string {
	r.lock.Lock()
	defer r.lock.Unlock()

	names := generics.Set[string]{}
	for n := range r.hasher {
		names.Add(n)
	}
	for _, p := range r.parents {
		if p == nil {
			continue
		}
		names.Add(p.HasherNames()...)
	}
	result := names.AsArray()
	sort.Strings(result)
	return result
}

func (r *hasherRegistry) GetHasher(name string) Hasher {
	r.lock.RLock()
	defer r.lock.RUnlock()

	h := r.hasher[NormalizeHashAlgorithm(name)]
	if h != nil {
		return h
	}
	for _, p := range r.parents {
		if p == nil {
			continue
		}
		h = p.GetHasher(name)
		if h != nil {
			return h
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

var defaultHandlerRegistry = NewHandlerRegistry()

func DefaultHandlerRegistry() HandlerRegistry {
	return defaultHandlerRegistry
}

////////////////////////////////////////////////////////////////////////////////

type keyRegistry struct {
	lock        sync.RWMutex
	parents     []KeyRegistry
	publicKeys  map[string]interface{}
	privateKeys map[string]interface{}
}

var _ KeyRegistry = (*keyRegistry)(nil)

func NewKeyRegistry(parents ...KeyRegistry) KeyRegistry {
	return &keyRegistry{
		parents:     slices.Clone(parents),
		publicKeys:  map[string]interface{}{},
		privateKeys: map[string]interface{}{},
	}
}

func (r *keyRegistry) Copy() KeyRegistry {
	return &keyRegistry{
		parents:     r.parents,
		publicKeys:  maps.Clone(r.publicKeys),
		privateKeys: maps.Clone(r.privateKeys),
	}
}

func (r *keyRegistry) HasKeys() bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	if len(r.publicKeys) > 0 || len(r.privateKeys) > 0 {
		return true
	}
	for _, p := range r.parents {
		if p == nil {
			continue
		}
		if p.HasKeys() {
			return true
		}
	}
	return false
}

func (r *keyRegistry) RegisterPublicKey(name string, key interface{}) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.publicKeys[name] = key
}

func (r *keyRegistry) RegisterPrivateKey(name string, key interface{}) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.privateKeys[name] = key
}

func (r *keyRegistry) GetPublicKey(name string) interface{} {
	r.lock.RLock()
	defer r.lock.RUnlock()
	k := r.publicKeys[name]
	if k != nil {
		return k
	}
	for _, p := range r.parents {
		if p == nil {
			continue
		}
		k = p.GetPublicKey(name)
		if k != nil {
			return k
		}
	}
	return nil
}

func (r *keyRegistry) GetPrivateKey(name string) interface{} {
	r.lock.RLock()
	defer r.lock.RUnlock()

	k := r.privateKeys[name]
	if k != nil {
		return k
	}
	for _, p := range r.parents {
		if p == nil {
			continue
		}
		k = p.GetPrivateKey(name)
		if k != nil {
			return k
		}
	}
	return nil
}

var defaultKeyRegistry = NewKeyRegistry()

func DefaultKeyRegistry() KeyRegistry {
	return defaultKeyRegistry
}

////////////////////////////////////////////////////////////////////////////////

type (
	_HandlerRegistry = HandlerRegistry
	_KeyRegistry     = KeyRegistry
)

type registry struct {
	_HandlerRegistry
	_KeyRegistry
}

var _ Registry = (*registry)(nil)

func NewRegistry(h HandlerRegistry, k KeyRegistry) Registry {
	return &registry{
		_HandlerRegistry: NewHandlerRegistry(h),
		_KeyRegistry:     NewKeyRegistry(k),
	}
}

func (r *registry) HandlerRegistry() HandlerRegistry {
	return r._HandlerRegistry
}

func (r *registry) KeyRegistry() KeyRegistry {
	return r._KeyRegistry
}

func (r *registry) Copy() Registry {
	return &registry{
		_HandlerRegistry: r.HandlerRegistry().Copy(),
		_KeyRegistry:     r.KeyRegistry().Copy(),
	}
}

func RegistryWithPreferredKeys(reg Registry, keys KeyRegistry) Registry {
	if keys == nil {
		return reg
	}
	return &registry{
		_HandlerRegistry: reg.HandlerRegistry(),
		_KeyRegistry:     NewKeyRegistry(keys, reg.KeyRegistry()),
	}
}

var defaultRegistry = NewRegistry(DefaultHandlerRegistry(), DefaultKeyRegistry())

func DefaultRegistry() Registry {
	return defaultRegistry
}
