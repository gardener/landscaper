// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package download

import (
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/utils/registry"
	"github.com/open-component-model/ocm/pkg/errors"
)

const ALL = "*"

type Handler interface {
	Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error)
}

type Registry interface {
	Register(arttype, mediatype string, hdlr Handler)
	Handler
	DownloadAsBlob(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error)
}

type _registry struct {
	lock     sync.RWMutex
	handlers *registry.Registry[Handler, registry.RegistrationKey]
}

func NewRegistry() Registry {
	return &_registry{
		handlers: registry.NewRegistry[Handler, registry.RegistrationKey](),
	}
}

func (r *_registry) Register(arttype, mediatype string, hdlr Handler) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.handlers.Register(registry.RegistrationKey{arttype, mediatype}, hdlr)
}

func (r *_registry) getHandlers(arttype, mediatype string) []Handler {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.handlers.LookupHandler(registry.RegistrationKey{arttype, mediatype})
}

func (r *_registry) Download(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	art := racc.Meta().GetType()
	m, err := racc.AccessMethod()
	if err != nil {
		return false, "", err
	}
	defer m.Close()
	mime := m.MimeType()
	if ok, p, err := r.download(r.getHandlers(art, mime), p, racc, path, fs); ok {
		return ok, p, err
	}
	return r.download(r.getHandlers(ALL, ""), p, racc, path, fs)
}

func (r *_registry) DownloadAsBlob(p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	return r.download(r.getHandlers(ALL, ""), p, racc, path, fs)
}

func (r *_registry) download(list []Handler, p common.Printer, racc cpi.ResourceAccess, path string, fs vfs.FileSystem) (bool, string, error) {
	errs := errors.ErrListf("download")
	for _, h := range list {
		ok, p, err := h.Download(p, racc, path, fs)
		if ok {
			return ok, p, err
		}
		errs.Add(err)
	}
	return false, "", errs.Result()
}

var DefaultRegistry = NewRegistry()

func RegisterForArtifactType(arttype string, hdlr Handler) {
	DefaultRegistry.Register(arttype, "", hdlr)
}

func Register(arttype, mediatype string, hdlr Handler) {
	DefaultRegistry.Register(arttype, mediatype, hdlr)
}

////////////////////////////////////////////////////////////////////////////////

const ATTR_DOWNLOADER_HANDLERS = "github.com/open-component-model/ocm/pkg/contexts/ocm/download"

func For(ctx datacontext.Context) Registry {
	if ctx == nil {
		return DefaultRegistry
	}
	return ctx.GetAttributes().GetAttribute(ATTR_DOWNLOADER_HANDLERS, DefaultRegistry).(Registry)
}

func SetFor(ctx datacontext.Context, registry Registry) {
	ctx.GetAttributes().SetAttribute(ATTR_DOWNLOADER_HANDLERS, registry)
}
