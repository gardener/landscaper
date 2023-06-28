// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"sort"
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/utils/registry"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
	"github.com/open-component-model/ocm/pkg/registrations"
	"github.com/open-component-model/ocm/pkg/utils"
)

const ALL = "*"

type Handler interface {
	Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error)
}

const DEFAULT_BLOBHANDLER_PRIO = 100

type PrioHandler struct {
	Handler
	Prio int
}

// MultiHandler is a Handler consisting of a sequence of handlers.
type MultiHandler []Handler

var _ sort.Interface = MultiHandler(nil)

func (m MultiHandler) Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	errs := errors.ErrListf("download")
	for _, h := range m {
		ok, p, err := h.Download(p, racc, path, fs)
		if ok {
			return ok, p, err
		}
		errs.Add(err)
	}
	return false, "", errs.Result()
}

func (m MultiHandler) Len() int {
	return len(m)
}

func (m MultiHandler) Less(i, j int) bool {
	pi := DEFAULT_BLOBHANDLER_PRIO
	pj := DEFAULT_BLOBHANDLER_PRIO

	if p, ok := m[i].(*PrioHandler); ok {
		pi = p.Prio
	}
	if p, ok := m[j].(*PrioHandler); ok {
		pj = p.Prio
	}
	return pi > pj
}

func (m MultiHandler) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

type Registry interface {
	registrations.HandlerRegistrationRegistry[Target, HandlerOption]

	Register(hdlr Handler, olist ...HandlerOption)
	LookupHandler(art, media string) MultiHandler
	Handler
	DownloadAsBlob(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error)
}

type _registry struct {
	registrations.HandlerRegistrationRegistry[Target, HandlerOption]

	id       finalizer.ObjectIdentity
	lock     sync.RWMutex
	base     Registry
	handlers *registry.Registry[Handler, registry.RegistrationKey]
}

func NewRegistry(base ...Registry) Registry {
	b := utils.Optional(base...)
	return &_registry{
		id:                          finalizer.NewObjectIdentity("downloader.registry.ocm.software"),
		HandlerRegistrationRegistry: NewHandlerRegistrationRegistry(b),
		base:                        b,
		handlers:                    registry.NewRegistry[Handler, registry.RegistrationKey](),
	}
}

func (r *_registry) LookupHandler(art, media string) MultiHandler {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return r.getHandlers(art, media)
}

func (r *_registry) Register(hdlr Handler, olist ...HandlerOption) {
	opts := NewHandlerOptions(olist...)
	r.lock.Lock()
	defer r.lock.Unlock()
	if opts.Priority != 0 {
		hdlr = &PrioHandler{hdlr, opts.Priority}
	}
	r.handlers.Register(registry.RegistrationKey{opts.ArtifactType, opts.MimeType}, hdlr)
}

func (r *_registry) getHandlers(arttype, mediatype string) MultiHandler {
	list := r.handlers.LookupHandler(registry.RegistrationKey{arttype, mediatype})
	if r.base != nil {
		list = append(list, r.base.LookupHandler(arttype, mediatype)...)
	}
	return list
}

func (r *_registry) Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	p = common.AssurePrinter(p)
	art := racc.Meta().GetType()
	m, err := racc.AccessMethod()
	if err != nil {
		return false, "", err
	}
	defer m.Close()
	mime := m.MimeType()
	if ok, p, err := r.download(r.LookupHandler(art, mime), p, racc, path, fs); ok {
		return ok, p, err
	}
	return r.download(r.LookupHandler(ALL, ""), p, racc, path, fs)
}

func (r *_registry) DownloadAsBlob(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	return r.download(r.LookupHandler(ALL, ""), p, racc, path, fs)
}

func (r *_registry) download(list MultiHandler, p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	sort.Stable(list)
	return list.Download(p, racc, path, fs)
}

var DefaultRegistry = NewRegistry()

func Register(hdlr Handler, olist ...HandlerOption) {
	DefaultRegistry.Register(hdlr, olist...)
}

////////////////////////////////////////////////////////////////////////////////

const ATTR_DOWNLOADER_HANDLERS = "github.com/open-component-model/ocm/pkg/contexts/ocm/download"

func For(ctx cpi.ContextProvider) Registry {
	if ctx == nil {
		return DefaultRegistry
	}
	return ctx.OCMContext().GetAttributes().GetOrCreateAttribute(ATTR_DOWNLOADER_HANDLERS, create).(Registry)
}

func create(datacontext.Context) interface{} {
	return NewRegistry(DefaultRegistry)
}

func SetFor(ctx datacontext.Context, registry Registry) {
	ctx.GetAttributes().SetAttribute(ATTR_DOWNLOADER_HANDLERS, registry)
}
