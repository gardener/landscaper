// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package helm

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/mandelsoft/vfs/pkg/vfs"
	"helm.sh/helm/v3/pkg/registry"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/errors"
)

const (
	ChartMediaType      = registry.ChartLayerMediaType
	ProvenanceMediaType = registry.ProvLayerMediaType
)

type ChartAccess interface {
	io.Closer
	Chart() (accessio.TemporaryBlobAccess, error)
	Prov() (accessio.TemporaryBlobAccess, error)
	ArtefactSet() (accessio.TemporaryBlobAccess, error)
}

func newFileAccess(c *chartAccess, path string, mime string) accessio.TemporaryBlobAccess {
	c.refcnt++
	return accessio.ReferencingBlobAccess(accessio.BlobAccessForFile(mime, path, c.fs), c.unref)
}

type chartAccess struct {
	lock sync.Mutex

	closed bool
	refcnt int

	fs    vfs.FileSystem
	root  string
	chart string
	prov  string
	aset  string
}

var _ ChartAccess = (*chartAccess)(nil)

func newTempChartAccess(fss ...vfs.FileSystem) (*chartAccess, error) {
	fs := accessio.FileSystem(fss...)

	temp, err := vfs.TempDir(fs, "", "helmchart")
	if err != nil {
		return nil, err
	}
	return &chartAccess{
		fs:   fs,
		root: temp,
	}, nil
}

func NewChartAccessByFiles(chart, prov string, fss ...vfs.FileSystem) ChartAccess {
	return &chartAccess{
		fs:    accessio.FileSystem(fss...),
		chart: chart,
		prov:  prov,
	}
}

func (c *chartAccess) unref() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.refcnt == 0 {
		return fmt.Errorf("oops: refcount is already zero")
	}
	c.refcnt--
	return nil
}

func (c *chartAccess) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.refcnt > 0 {
		return errors.ErrStillInUse("chart access")
	}

	defer func() { c.closed = true }()

	if c.root != "" && !c.closed {
		return os.RemoveAll(c.root)
	}
	return nil
}

func (c *chartAccess) Chart() (accessio.TemporaryBlobAccess, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return nil, accessio.ErrClosed
	}

	return newFileAccess(c, c.chart, ChartMediaType), nil
}

func (c *chartAccess) Prov() (accessio.TemporaryBlobAccess, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return nil, accessio.ErrClosed
	}
	if c.prov == "" {
		return nil, nil
	}
	return newFileAccess(c, c.prov, ProvenanceMediaType), nil
}

func (c *chartAccess) ArtefactSet() (accessio.TemporaryBlobAccess, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return nil, accessio.ErrClosed
	}
	if c.aset == "" {
		return nil, nil
	}
	return newFileAccess(c, c.aset, artdesc.MediaTypeImageManifest), nil
}
