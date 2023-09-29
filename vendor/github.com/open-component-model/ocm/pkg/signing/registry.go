// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"sort"
	"sync"
)

type Registry interface {
	HandlerRegistry
	KeyRegistry
}

type HandlerRegistry interface {
	RegisterSignatureHandler(handler SignatureHandler)
	RegisterSigner(algo string, signer Signer)
	RegisterVerifier(algo string, verifier Verifier)
	GetSigner(name string) Signer
	GetVerifier(name string) Verifier
	SignerNames() []string

	RegisterHasher(hasher Hasher)
	GetHasher(name string) Hasher
	HasherNames() []string
}

type KeyRegistry interface {
	RegisterPublicKey(name string, key interface{})
	RegisterPrivateKey(name string, key interface{})
	GetPublicKey(name string) interface{}
	GetPrivateKey(name string) interface{}
}

////////////////////////////////////////////////////////////////////////////////

type handlerRegistry struct {
	lock     sync.RWMutex
	signers  map[string]Signer
	verifier map[string]Verifier
	hasher   map[string]Hasher
}

var _ HandlerRegistry = (*handlerRegistry)(nil)

func NewHandlerRegistry() HandlerRegistry {
	return &handlerRegistry{
		signers:  map[string]Signer{},
		verifier: map[string]Verifier{},
		hasher:   map[string]Hasher{},
	}
}

func (r *handlerRegistry) RegisterSignatureHandler(handler SignatureHandler) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.signers[handler.Algorithm()] = handler
	r.verifier[handler.Algorithm()] = handler
}

func (r *handlerRegistry) RegisterSigner(algo string, signer Signer) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.signers[algo] = signer
	if v, ok := signer.(Verifier); ok && r.verifier[algo] == nil {
		r.verifier[algo] = v
	}
}

func (r *handlerRegistry) SignerNames() []string {
	r.lock.Lock()
	defer r.lock.Unlock()
	names := []string{}
	for n := range r.signers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (r *handlerRegistry) RegisterVerifier(algo string, verifier Verifier) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.verifier[algo] = verifier
	if v, ok := verifier.(Signer); ok && r.signers[algo] == nil {
		r.signers[algo] = v
	}
}

func (r *handlerRegistry) RegisterHasher(hasher Hasher) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.hasher[hasher.Algorithm()] = hasher
}

func (r *handlerRegistry) HasherNames() []string {
	r.lock.Lock()
	defer r.lock.Unlock()
	names := []string{}
	for n := range r.hasher {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (r *handlerRegistry) GetSigner(name string) Signer {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.signers[name]
}

func (r *handlerRegistry) GetVerifier(name string) Verifier {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.verifier[name]
}

func (r *handlerRegistry) GetHasher(name string) Hasher {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.hasher[NormalizeHashAlgorithm(name)]
}

////////////////////////////////////////////////////////////////////////////////

var defaultHandlerRegistry = NewHandlerRegistry()

func DefaultHandlerRegistry() HandlerRegistry {
	return defaultHandlerRegistry
}

////////////////////////////////////////////////////////////////////////////////

type keyRegistry struct {
	lock        sync.RWMutex
	publicKeys  map[string]interface{}
	privateKeys map[string]interface{}
}

var _ KeyRegistry = (*keyRegistry)(nil)

func NewKeyRegistry() KeyRegistry {
	return &keyRegistry{
		publicKeys:  map[string]interface{}{},
		privateKeys: map[string]interface{}{},
	}
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
	return r.publicKeys[name]
}

func (r *keyRegistry) GetPrivateKey(name string) interface{} {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.privateKeys[name]
}

var defaultKeyRegistry = NewKeyRegistry()

func DefaultKeyRegistry() KeyRegistry {
	return defaultKeyRegistry
}

////////////////////////////////////////////////////////////////////////////////

type registry struct {
	baseHandlers HandlerRegistry
	baseKeys     KeyRegistry
	handlers     HandlerRegistry
	keys         KeyRegistry
}

var _ Registry = (*registry)(nil)

func NewRegistry(h HandlerRegistry, k KeyRegistry) Registry {
	return &registry{
		baseHandlers: h,
		baseKeys:     k,
		handlers:     NewHandlerRegistry(),
		keys:         NewKeyRegistry(),
	}
}

func (r *registry) RegisterSignatureHandler(handler SignatureHandler) {
	r.handlers.RegisterSignatureHandler(handler)
}

func (r *registry) RegisterSigner(algo string, signer Signer) {
	r.handlers.RegisterSigner(algo, signer)
}

func (r *registry) RegisterVerifier(algo string, verifier Verifier) {
	r.handlers.RegisterVerifier(algo, verifier)
}

func (r *registry) SignerNames() []string {
	names := r.baseHandlers.SignerNames()
outer:
	for _, n := range r.handlers.SignerNames() {
		for _, e := range names {
			if e == n {
				continue outer
			}
		}
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (r *registry) GetSigner(name string) Signer {
	s := r.handlers.GetSigner(name)
	if s == nil && r.baseHandlers != nil {
		s = r.baseHandlers.GetSigner(name)
	}
	return s
}

func (r *registry) GetVerifier(name string) Verifier {
	s := r.handlers.GetVerifier(name)
	if s == nil && r.baseHandlers != nil {
		s = r.baseHandlers.GetVerifier(name)
	}
	return s
}

func (r *registry) RegisterHasher(hasher Hasher) {
	r.handlers.RegisterHasher(hasher)
}

func (r *registry) GetHasher(name string) Hasher {
	s := r.handlers.GetHasher(NormalizeHashAlgorithm(name))
	if s == nil && r.baseHandlers != nil {
		s = r.baseHandlers.GetHasher(name)
	}
	return s
}

func (r *registry) HasherNames() []string {
	names := r.baseHandlers.HasherNames()
outer:
	for _, n := range r.handlers.HasherNames() {
		for _, e := range names {
			if e == n {
				continue outer
			}
		}
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (r *registry) RegisterPublicKey(name string, key interface{}) {
	r.keys.RegisterPublicKey(name, key)
}

func (r *registry) RegisterPrivateKey(name string, key interface{}) {
	r.keys.RegisterPrivateKey(name, key)
}

func (r *registry) GetPublicKey(name string) interface{} {
	s := r.keys.GetPublicKey(name)
	if s == nil && r.baseKeys != nil {
		s = r.baseKeys.GetPublicKey(name)
	}
	return s
}

func (r *registry) GetPrivateKey(name string) interface{} {
	s := r.keys.GetPrivateKey(name)
	if s == nil && r.baseKeys != nil {
		s = r.baseKeys.GetPrivateKey(name)
	}
	return s
}

var defaultRegistry = NewRegistry(DefaultHandlerRegistry(), DefaultKeyRegistry())

func DefaultRegistry() Registry {
	return defaultRegistry
}
