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

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/common/accessobj"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/attrs/vfsattr"
	"github.com/open-component-model/ocm/pkg/contexts/ocm/cpi/accspeccpi"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/iotools"
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
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](Type, accspeccpi.WithDescription(usage)))
	accspeccpi.RegisterAccessType(accspeccpi.NewAccessSpecType[*AccessSpec](TypeV1, accspeccpi.WithFormatSpec(formatV1), accspeccpi.WithConfigHandler(ConfigHandler())))
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

var _ accspeccpi.AccessSpec = (*AccessSpec)(nil)

// New creates a new NPM registry access spec version v1.
func New(registry, pkg, version string) *AccessSpec {
	return &AccessSpec{
		ObjectVersionedType: runtime.NewVersionedTypedObject(Type),
		Registry:            registry,
		Package:             pkg,
		Version:             version,
	}
}

func (a *AccessSpec) Describe(ctx accspeccpi.Context) string {
	return fmt.Sprintf("NPM package %s:%s in registry %s", a.Package, a.Version, a.Registry)
}

func (_ *AccessSpec) IsLocal(accspeccpi.Context) bool {
	return false
}

func (a *AccessSpec) GlobalAccessSpec(ctx accspeccpi.Context) accspeccpi.AccessSpec {
	return a
}

func (a *AccessSpec) GetReferenceHint(cv accspeccpi.ComponentVersionAccess) string {
	return a.Package + ":" + a.Version
}

func (_ *AccessSpec) GetType() string {
	return Type
}

func (a *AccessSpec) AccessMethod(c accspeccpi.ComponentVersionAccess) (accspeccpi.AccessMethod, error) {
	return accspeccpi.AccessMethodForImplementation(newMethod(c, a))
}

func (a *AccessSpec) GetInexpensiveContentVersionIdentity(access accspeccpi.ComponentVersionAccess) string {
	meta, _ := a.getPackageMeta(access.GetContext())
	if meta != nil {
		return meta.Dist.Shasum
	}
	return ""
}

func (a *AccessSpec) getPackageMeta(ctx accspeccpi.Context) (*meta, error) {
	url := a.Registry + path.Join("/", a.Package, a.Version)
	r, err := reader(url, vfsattr.Get(ctx))
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, io.LimitReader(r, 200000))
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get version metadata for %s", url)
	}

	var metadata meta

	err = json.Unmarshal(buf.Bytes(), &metadata)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshal version metadata for %s", url)
	}
	return &metadata, nil
}

////////////////////////////////////////////////////////////////////////////////

func newMethod(c accspeccpi.ComponentVersionAccess, a *AccessSpec) (accspeccpi.AccessMethodImpl, error) {
	factory := func() (blobaccess.BlobAccess, error) {
		meta, err := a.getPackageMeta(c.GetContext())
		if err != nil {
			return nil, err
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
				return iotools.VerifyingReaderWithHash(r, crypto.SHA1, meta.Dist.Shasum), nil
			}
		}
		acc := blobaccess.DataAccessForReaderFunction(f, meta.Dist.Tarball)
		return accessobj.CachedBlobAccessForWriter(c.GetContext(), mime.MIME_TGZ, accessio.NewDataAccessWriter(acc)), nil
	}
	return accspeccpi.NewDefaultMethodImpl(c, a, "", mime.MIME_TGZ, factory), nil
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
