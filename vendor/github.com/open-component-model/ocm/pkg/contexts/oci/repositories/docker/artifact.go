// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"sync"

	"github.com/containers/image/v5/types"
	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
)

type dockerSource struct {
	lock     sync.RWMutex
	src      types.ImageSource
	img      types.Image
	refcount int
}

var _ accessio.BlobSource = (*dockerSource)(nil)

func newDockerSource(img types.Image, src types.ImageSource) *dockerSource {
	return &dockerSource{
		src:      src,
		img:      img,
		refcount: 1,
	}
}

func (c *dockerSource) Ref() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.refcount == 0 {
		return accessio.ErrClosed
	}
	c.refcount++
	return nil
}

func (c *dockerSource) Unref() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.refcount == 0 {
		return accessio.ErrClosed
	}
	c.refcount--
	return c.src.Close()
}

func (d *dockerSource) GetBlobData(digest digest.Digest) (int64, blobaccess.DataAccess, error) {
	info := d.img.ConfigInfo()
	if info.Digest == digest {
		data, err := d.img.ConfigBlob(dummyContext)
		if err != nil {
			return -1, nil, err
		}
		return info.Size, blobaccess.DataAccessForBytes(data), nil
	}
	info.Digest = ""
	for _, l := range d.img.LayerInfos() {
		if l.Digest == digest {
			info = l
			acc, err := NewDataAccess(d.src, info, false)
			return l.Size, acc, err
		}
	}
	return -1, nil, cpi.ErrBlobNotFound(digest)
}
