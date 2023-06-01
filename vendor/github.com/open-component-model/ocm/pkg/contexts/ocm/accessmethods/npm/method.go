// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package npm

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/mime"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// TODO: open questions
// - authentication???
// - writing packages

// Type is the access type of NPM registry.
const (
	Type   = "npm"
	TypeV1 = Type + runtime.VersionSeparator + "v1"
)

func init() {
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](Type, cpi.WithDescription(usage)))
	cpi.RegisterAccessType(cpi.NewAccessSpecType[*AccessSpec](TypeV1, cpi.WithFormatSpec(formatV1), cpi.WithConfigHandler(ConfigHandler())))
}

// AccessSpec describes the access for a NPM registry.
type AccessSpec struct {
	runtime.ObjectVersionedType `json:",inline"`

	// Registry is the base URL of the NPM registry
	Registry string `json:"registry"`
	// Package is the name of NPM package
	Package string `json:"package"`
	// Version of the NPM package.
	Version string `json:"version"`
}

var _ cpi.AccessSpec = (*AccessSpec)(nil)

// New creates a new NPM registry access spec version v1.
func New(registry, pkg, version string) *AccessSpec {
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		Registry:            registry,
		Package:             pkg,
		Version:             version,
	}
}

func (a *AccessSpec) Describe(ctx cpi.Context) string {
	return fmt.Sprintf("NPM package %s:%s in registry %s", a.Package, a.Version, a.Registry)
}

func (_ *AccessSpec) IsLocal(cpi.Context) bool {
	return false
}

func (a *AccessSpec) GlobalAccessSpec(ctx cpi.Context) cpi.AccessSpec {
	return a
}

func (a *AccessSpec) GetReferenceHint(cv cpi.ComponentVersionAccess) string {
	return a.Package + ":" + a.Version
}

func (_ *AccessSpec) GetType() string {
	return Type
}

func (a *AccessSpec) AccessMethod(c cpi.ComponentVersionAccess) (cpi.AccessMethod, error) {
	return newMethod(c, a)
}

////////////////////////////////////////////////////////////////////////////////

func newMethod(c cpi.ComponentVersionAccess, a *AccessSpec) (cpi.AccessMethod, error) {
	factory := func() (accessio.BlobAccess, error) {
		url := a.Registry + path.Join("/", a.Package, a.Version)
		r, err := reader(url, vfsattr.Get(c.GetContext()))
		if err != nil {
			return nil, err
		}
		buf := &bytes.Buffer{}
		_, err = io.Copy(buf, io.LimitReader(r, 200000))
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get version metadata for %s", url)
		}

		var meta meta

		err = json.Unmarshal(buf.Bytes(), &meta)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal version metadata for %s", url)
		}

		f := func() (io.ReadCloser, error) {
			return reader(meta.Dist.Tarball, vfsattr.Get(c.GetContext()))
		}
		if meta.Dist.Shasum != "" {
			tf := f
			f = func() (io.ReadCloser, error) {
				r, err := tf()
				if err != nil {
					return nil, err
				}
				return accessio.VerifyingReaderWithHash(r, crypto.SHA1, meta.Dist.Shasum), nil
			}
		}
		acc := accessio.DataAccessForReaderFunction(f, meta.Dist.Tarball)
		return accessobj.CachedBlobAccessForWriter(c.GetContext(), mime.MIME_TGZ, accessio.NewDataAccessWriter(acc)), nil
	}
	return cpi.NewDefaultMethod(c, a, mime.MIME_TGZ, factory), nil
}

type meta struct {
	Dist struct {
		Shasum  string `json:"shasum"`
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

func reader(url string, fs vfs.FileSystem) (io.ReadCloser, error) {
	c := &http.Client{}

	if strings.HasPrefix(url, "file://") {
		path := url[7:]
		return fs.OpenFile(path, vfs.O_RDONLY, 0o600)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		buf := &bytes.Buffer{}
		_, err = io.Copy(buf, io.LimitReader(resp.Body, 2000))
		if err != nil {
			return nil, errors.Newf("version meta data request %s provides %s", url, resp.Status)
		}
		return nil, errors.Newf("version meta data request %s provides %s: %s", url, resp.Status, buf.String())
	}
	return resp.Body, nil
}
