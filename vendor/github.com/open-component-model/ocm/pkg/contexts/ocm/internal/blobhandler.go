// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

type ImplementationRepositoryType struct {
	ContextType    string
	RepositoryType string
}

func (t ImplementationRepositoryType) String() string {
	return fmt.Sprintf("%s[%s]", t.RepositoryType, t.ContextType)
}

func (t ImplementationRepositoryType) IsInitial() bool {
	return t.RepositoryType == "" && t.ContextType == ""
}

// StorageContext is an object describing the storage context used for the
// mapping of a component repository to a base repository (e.g. oci api)
// It depends on the Context type of the used base repository.
type StorageContext interface {
	GetContext() Context
	TargetComponentVersion() ComponentVersionAccess
	TargetComponentRepository() Repository
	GetImplementationRepositoryType() ImplementationRepositoryType
}

// BlobHandler s the interface for a dedicated handling of storing blobs
// for the LocalBlob access method in a dedicated kind of repository.
// with the possibility of access by an external distribution spec.
// (besides of the blob storage as part of a component version).
// The technical repository to use should be derivable from the chosen
// component directory or passed together with the storage context.
// The task of the handler is to store the local blob on its own
// responsibility and to return an appropriate global access method.
type BlobHandler interface {
	// StoreBlob has the chance to decide to store a local blob
	// in a repository specific fashion providing external access.
	// If this is possible and done an appropriate access spec
	// must be returned, if this is not done, nil has to be returned
	// without error
	StoreBlob(blob BlobAccess, artType, hint string, global AccessSpec, ctx StorageContext) (AccessSpec, error)
}

// MultiBlobHandler is a BlobHandler consisting of a sequence of handlers.
type MultiBlobHandler []BlobHandler

var _ sort.Interface = MultiBlobHandler(nil)

func (m MultiBlobHandler) StoreBlob(blob BlobAccess, artType, hint string, global AccessSpec, ctx StorageContext) (AccessSpec, error) {
	for _, h := range m {
		a, err := h.StoreBlob(blob, artType, hint, global, ctx)
		if err != nil {
			return nil, err
		}
		if a != nil {
			return a, nil
		}
	}
	return nil, nil
}

func (m MultiBlobHandler) Len() int {
	return len(m)
}

func (m MultiBlobHandler) Less(i, j int) bool {
	pi := DEFAULT_BLOBHANDLER_PRIO
	pj := DEFAULT_BLOBHANDLER_PRIO

	if p, ok := m[i].(*PrioBlobHandler); ok {
		pi = p.Prio
	}
	if p, ok := m[j].(*PrioBlobHandler); ok {
		pj = p.Prio
	}
	return pi > pj
}

func (m MultiBlobHandler) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

////////////////////////////////////////////////////////////////////////////////

type BlobHandlerOptions struct {
	BlobHandlerKey
	Priority int
}

func NewBlobHandlerOptions(olist ...BlobHandlerOption) *BlobHandlerOptions {
	var opts BlobHandlerOptions
	for _, o := range olist {
		o.ApplyBlobHandlerOptionTo(&opts)
	}
	return &opts
}

func (o BlobHandlerOptions) ApplyBlobHandlerOptionTo(opts *BlobHandlerOptions) {
	if o.Priority > 0 {
		opts.Priority = o.Priority
	}
	o.BlobHandlerKey.ApplyBlobHandlerOptionTo(opts)
}

type BlobHandlerOption interface {
	ApplyBlobHandlerOptionTo(*BlobHandlerOptions)
}

type prio struct {
	prio int
}

func WithPrio(p int) BlobHandlerOption {
	return prio{p}
}

func (o prio) ApplyBlobHandlerOptionTo(opts *BlobHandlerOptions) {
	opts.Priority = o.prio
}

////////////////////////////////////////////////////////////////////////////////

// BlobHandlerKey is the registration key for BlobHandlers.
type BlobHandlerKey struct {
	ImplementationRepositoryType
	ArtifactType string
	MimeType     string
}

var _ BlobHandlerOption = BlobHandlerKey{}

func NewBlobHandlerKey(ctxtype, repotype, artifactType, mimetype string) BlobHandlerKey {
	return BlobHandlerKey{
		ImplementationRepositoryType: ImplementationRepositoryType{
			ContextType:    ctxtype,
			RepositoryType: repotype,
		},
		ArtifactType: artifactType,
		MimeType:     mimetype,
	}
}

func (k BlobHandlerKey) ApplyBlobHandlerOptionTo(opts *BlobHandlerOptions) {
	if k.ContextType != "" {
		opts.ContextType = k.ContextType
	}
	if k.RepositoryType != "" {
		opts.RepositoryType = k.RepositoryType
	}
	if k.ArtifactType != "" {
		opts.ArtifactType = k.ArtifactType
	}
	if k.MimeType != "" {
		opts.MimeType = k.MimeType
	}
}

func ForRepo(ctxtype, repotype string) BlobHandlerOption {
	return BlobHandlerKey{ImplementationRepositoryType: ImplementationRepositoryType{ContextType: ctxtype, RepositoryType: repotype}}
}

func ForMimeType(mimetype string) BlobHandlerOption {
	return BlobHandlerKey{MimeType: mimetype}
}

func ForArtifactType(artifacttype string) BlobHandlerOption {
	return BlobHandlerKey{ArtifactType: artifacttype}
}

////////////////////////////////////////////////////////////////////////////////

type BlobHandlerConfig interface{}

type BlobHandlerRegistrationHandler interface {
	RegisterByName(handler string, ctx Context, config BlobHandlerConfig, opts ...BlobHandlerOption) (bool, error)
}

type BlobHandlerRegistrationRegistry interface {
	BlobHandlerRegistrationHandler
	RegisterRegistrationHandler(path string, handler BlobHandlerRegistrationHandler)
	GetRegistrationHandlers(name string) []*RegistrationHandlerInfo
}

type NamePath []string

func NewNamePath(path string) NamePath {
	return strings.Split(path, "/")
}

func (p NamePath) Compare(o NamePath) int {
	if d := len(p) - len(o); d != 0 {
		return d
	}
	for i, e := range p {
		if d := strings.Compare(e, o[i]); d != 0 {
			return d
		}
	}
	return 0
}

func (p NamePath) IsPrefixOf(o NamePath) bool {
	if len(p) > len(o) {
		return false
	}
	for i, e := range p {
		if e != o[i] {
			return false
		}
	}
	return true
}

type RegistrationHandlerInfo struct {
	prefix  NamePath
	handler BlobHandlerRegistrationHandler
}

func NewRegistrationHandlerInfo(path string, handler BlobHandlerRegistrationHandler) *RegistrationHandlerInfo {
	return &RegistrationHandlerInfo{
		prefix:  NewNamePath(path),
		handler: handler,
	}
}

func (i *RegistrationHandlerInfo) RegisterByName(handler string, ctx Context, config BlobHandlerConfig, opts ...BlobHandlerOption) (bool, error) {
	path := NewNamePath(handler)

	if !i.prefix.IsPrefixOf(path) {
		return false, nil
	}
	return i.handler.RegisterByName(strings.Join(path[len(i.prefix):], "/"), ctx, config, opts...)
}

type handlerRegistrationRegistry struct {
	lock     sync.RWMutex
	base     BlobHandlerRegistrationRegistry
	handlers []*RegistrationHandlerInfo
}

func NewBlobHandlerRegistrationRegistry(base ...BlobHandlerRegistrationRegistry) BlobHandlerRegistrationRegistry {
	return &handlerRegistrationRegistry{base: utils.Optional(base...)}
}

func (c *handlerRegistrationRegistry) RegisterRegistrationHandler(path string, handler BlobHandlerRegistrationHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()

	comps := strings.Split(path, "/")
	n := &RegistrationHandlerInfo{
		prefix:  comps,
		handler: handler,
	}

	var i int
	var h *RegistrationHandlerInfo
	for i, h = range c.handlers {
		if h.prefix.Compare(comps) < 0 {
			break
		}
	}
	c.handlers = append(c.handlers[:i], append([]*RegistrationHandlerInfo{n}, c.handlers[i:]...)...)
}

func (c *handlerRegistrationRegistry) GetRegistrationHandlers(name string) []*RegistrationHandlerInfo {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var result []*RegistrationHandlerInfo
	path := NewNamePath(name)
	for _, h := range c.handlers {
		if h.prefix.IsPrefixOf(path) {
			result = append(result, h)
		}
	}

	if c.base != nil {
		base := c.base.GetRegistrationHandlers(name)
		i := 0
		for _, h := range base {
			for i != len(result) && result[i].prefix.Compare(h.prefix) >= 0 {
				i++
			}
			result = append(result[:i], append([]*RegistrationHandlerInfo{h}, result[i:]...)...)
			i++
		}
	}
	return result
}

func (c *handlerRegistrationRegistry) RegisterByName(handler string, ctx Context, config BlobHandlerConfig, opts ...BlobHandlerOption) (bool, error) {
	list := c.GetRegistrationHandlers(handler)
	errlist := errors.ErrListf("blob handler registration")
	for _, h := range list {
		ok, err := h.RegisterByName(handler, ctx, config, opts...)
		if ok {
			return ok, err
		}
		errlist.Add(err)
	}
	if errlist.Len() > 0 {
		return false, errlist.Result()
	}
	return false, fmt.Errorf("no registration handler found for %s", handler)
}

////////////////////////////////////////////////////////////////////////////////

// BlobHandlerRegistry registers blob handlers to use in a dedicated ocm context.
type BlobHandlerRegistry interface {
	BlobHandlerRegistrationRegistry

	IsInitial() bool

	// Copy provides a new independend copy of the registry.
	Copy() BlobHandlerRegistry
	// RegisterBlobHandler registers a blob handler. It must specify either a sole mime type,
	// or a context and repository type, or all three keys.
	Register(handler BlobHandler, opts ...BlobHandlerOption) BlobHandlerRegistry

	// GetHandler returns the handler with the given key.
	GetHandler(key BlobHandlerKey) BlobHandler

	// LookupHandler returns handler trying all matches in the following order:
	//
	// - a handler matching all keys
	// - handlers matching the repo and mime type (from specific to more general by discarding + components)
	//   - with artifact type
	//   - without artifact type
	// - handlers matching artifact type
	// - handlers matching a sole mimetype handler (from specific to more general by discarding + components)
	// - a handler matching the repo
	//
	LookupHandler(repotype ImplementationRepositoryType, artifacttype, mimeType string) BlobHandler
}

const DEFAULT_BLOBHANDLER_PRIO = 100

type PrioBlobHandler struct {
	BlobHandler
	Prio int
}

type handlerCache struct {
	cache map[BlobHandlerKey]BlobHandler
}

func newHandlerCache() *handlerCache {
	return &handlerCache{map[BlobHandlerKey]BlobHandler{}}
}

func (c *handlerCache) len() int {
	return len(c.cache)
}

func (c *handlerCache) get(key BlobHandlerKey) (BlobHandler, bool) {
	h, ok := c.cache[key]
	return h, ok
}

func (c *handlerCache) set(key BlobHandlerKey, h BlobHandler) {
	c.cache[key] = h
}

type registrationHandlers = BlobHandlerRegistrationRegistry

type blobHandlerRegistry struct {
	lock       sync.RWMutex
	base       BlobHandlerRegistry
	handlers   map[BlobHandlerKey]BlobHandler
	defhandler MultiBlobHandler

	registrationHandlers

	cache *handlerCache
}

var DefaultBlobHandlerRegistry = NewBlobHandlerRegistry()

func NewBlobHandlerRegistry(base ...BlobHandlerRegistry) BlobHandlerRegistry {
	b := utils.Optional(base...)
	return &blobHandlerRegistry{
		base:                 b,
		handlers:             map[BlobHandlerKey]BlobHandler{},
		registrationHandlers: NewBlobHandlerRegistrationRegistry(b),
		cache:                newHandlerCache(),
	}
}

func (r *blobHandlerRegistry) Copy() BlobHandlerRegistry {
	r.lock.RLock()
	defer r.lock.RUnlock()
	n := NewBlobHandlerRegistry(r.base).(*blobHandlerRegistry)
	n.defhandler = append(n.defhandler, r.defhandler...)
	for k, h := range r.handlers {
		n.handlers[k] = h
	}
	return n
}

func (r *blobHandlerRegistry) IsInitial() bool {
	if r.base != nil && !r.base.IsInitial() {
		return false
	}
	return len(r.handlers) == 0 && len(r.defhandler) == 0
}

func (r *blobHandlerRegistry) Register(handler BlobHandler, olist ...BlobHandlerOption) BlobHandlerRegistry {
	opts := NewBlobHandlerOptions(olist...)
	r.lock.Lock()
	defer r.lock.Unlock()

	def := BlobHandlerKey{}

	if opts.Priority != 0 {
		handler = &PrioBlobHandler{handler, opts.Priority}
	}
	if opts.BlobHandlerKey == def {
		r.defhandler = append(r.defhandler, handler)
	} else {
		r.handlers[opts.BlobHandlerKey] = handler
	}
	if r.cache.len() > 0 {
		r.cache = newHandlerCache()
	}
	return r
}

func (r *blobHandlerRegistry) forMimeType(ctxtype, repotype, artifacttype, mimetype string) MultiBlobHandler {
	var multi MultiBlobHandler

	mime := mimetype
	for {
		if h := r.getHandler(NewBlobHandlerKey(ctxtype, repotype, artifacttype, mime)); h != nil {
			multi = append(multi, h)
		}
		idx := strings.LastIndex(mime, "+")
		if idx < 0 {
			break
		}
		mime = mime[:idx]
	}
	return multi
}

func (r *blobHandlerRegistry) GetHandler(key BlobHandlerKey) BlobHandler {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.getHandler(key)
}

func (r *blobHandlerRegistry) getHandler(key BlobHandlerKey) BlobHandler {
	def := BlobHandlerKey{}

	if key == def {
		if len(r.defhandler) > 0 {
			return r.defhandler
		}
	}
	h := r.handlers[key]
	if h != nil {
		return h
	}
	if r.base != nil {
		return r.base.GetHandler(key)
	}
	return nil
}

func (r *blobHandlerRegistry) LookupHandler(repotype ImplementationRepositoryType, artifacttype, mimetype string) BlobHandler {
	key := BlobHandlerKey{
		ImplementationRepositoryType: repotype,
		ArtifactType:                 artifacttype,
		MimeType:                     mimetype,
	}
	h, cache := r.lookupHandler(key)
	if cache != nil {
		r.lock.Lock()
		defer r.lock.Unlock()
		// fill cache, if unchanged during pseudo lock upgrade (no support in go sync package for that).
		// if cache has been renewed in the meantime, just use the old outdated result, but don't update.
		if r.cache == cache {
			r.cache.set(key, h)
		}
	}
	return h
}

func (r *blobHandlerRegistry) lookupHandler(key BlobHandlerKey) (BlobHandler, *handlerCache) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if h, ok := r.cache.get(key); ok {
		return h, nil
	}
	var multi MultiBlobHandler
	if !key.ImplementationRepositoryType.IsInitial() {
		multi = append(multi, r.forMimeType(key.ContextType, key.RepositoryType, key.ArtifactType, key.MimeType)...)
		if key.MimeType != "" {
			multi = append(multi, r.forMimeType(key.ContextType, key.RepositoryType, key.ArtifactType, "")...)
		}
		if key.ArtifactType != "" {
			multi = append(multi, r.forMimeType(key.ContextType, key.RepositoryType, "", key.MimeType)...)
		}
	}
	multi = append(multi, r.forMimeType("", "", key.ArtifactType, key.MimeType)...)
	if key.MimeType != "" {
		multi = append(multi, r.forMimeType("", "", key.ArtifactType, "")...)
	}
	if key.ArtifactType != "" {
		multi = append(multi, r.forMimeType("", "", "", key.MimeType)...)
	}
	if !key.ImplementationRepositoryType.IsInitial() && key.ArtifactType != "" && key.MimeType != "" {
		multi = append(multi, r.forMimeType(key.ContextType, key.RepositoryType, "", "")...)
	}

	def := r.getHandler(BlobHandlerKey{})
	if def != nil {
		if m, ok := def.(MultiBlobHandler); ok {
			multi = append(multi, m...)
		} else {
			multi = append(multi, def)
		}
	}
	if len(multi) == 0 {
		return nil, r.cache
	}
	sort.Sort(multi)
	return multi, r.cache
}

func RegisterBlobHandler(handler BlobHandler, opts ...BlobHandlerOption) {
	DefaultBlobHandlerRegistry.Register(handler, opts...)
}

func MustRegisterBlobHandler(handler BlobHandler, opts ...BlobHandlerOption) {
	DefaultBlobHandlerRegistry.Register(handler, opts...)
}

func RegisterBlobHandlerRegistrationHandler(path string, handler BlobHandlerRegistrationHandler) {
	DefaultBlobHandlerRegistry.RegisterRegistrationHandler(path, handler)
}
