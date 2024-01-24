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

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/refmgmt"
	"github.com/open-component-model/ocm/pkg/utils"
)

const (
	ChartMediaType      = registry.ChartLayerMediaType
	ProvenanceMediaType = registry.ProvLayerMediaType
)

type ChartAccess interface {
	io.Closer
	Chart() (blobaccess.BlobAccess, error)
	Prov() (blobaccess.BlobAccess, error)
	ArtefactSet() (blobaccess.BlobAccess, error)
}

func newFileAccess(c *chartAccess, path string, mime string) blobaccess.BlobAccess {
	c.refcnt++
	return blobaccess.ForFileWithCloser(refmgmt.CloserFunc(c.unref), mime, path, c.fs)
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
	fs := utils.FileSystem(fss...)

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
		fs:    utils.FileSystem(fss...),
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
	if c.refcnt == 0 && c.closed {
		return c.cleanup()
	}
	return nil
}

func (c *chartAccess) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	defer func() { c.closed = true }()
	if c.refcnt == 0 {
		return c.cleanup()
	}
	return nil
}

func (c *chartAccess) cleanup() error {
	if c.root != "" {
		err := os.RemoveAll(c.root)
		c.root = ""
		return err
	}
	return nil
}

func (c *chartAccess) Chart() (blobaccess.BlobAccess, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return nil, blobaccess.ErrClosed
	}

	return newFileAccess(c, c.chart, ChartMediaType), nil
}

func (c *chartAccess) Prov() (blobaccess.BlobAccess, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return nil, blobaccess.ErrClosed
	}
	if c.prov == "" {
		return nil, nil
	}
	return newFileAccess(c, c.prov, ProvenanceMediaType), nil
}

func (c *chartAccess) ArtefactSet() (blobaccess.BlobAccess, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.closed {
		return nil, blobaccess.ErrClosed
	}
	if c.aset == "" {
		return nil, nil
	}
	return newFileAccess(c, c.aset, artdesc.MediaTypeImageManifest), nil
}
