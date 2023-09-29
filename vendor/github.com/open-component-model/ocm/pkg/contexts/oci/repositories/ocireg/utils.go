// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package ocireg

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/remotes"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/common/accessio"
	"github.com/open-component-model/ocm/pkg/contexts/oci/artdesc"
	"github.com/open-component-model/ocm/pkg/contexts/oci/cpi"
	"github.com/open-component-model/ocm/pkg/docker/resolve"
	"github.com/open-component-model/ocm/pkg/logging"
)

// TODO: add cache

type dataAccess struct {
	accessio.NopCloser
	lock    sync.Mutex
	fetcher remotes.Fetcher
	desc    artdesc.Descriptor
	reader  io.ReadCloser
}

var _ cpi.DataAccess = (*dataAccess)(nil)

func NewDataAccess(fetcher remotes.Fetcher, digest digest.Digest, mimeType string, delayed bool) (*dataAccess, error) {
	var reader io.ReadCloser
	var err error
	desc := artdesc.Descriptor{
		MediaType: mimeType,
		Digest:    digest,
		Size:      accessio.BLOB_UNKNOWN_SIZE,
	}
	if !delayed {
		reader, err = fetcher.Fetch(dummyContext, desc)
		if err != nil {
			return nil, err
		}
	}
	return &dataAccess{
		fetcher: fetcher,
		desc:    desc,
		reader:  reader,
	}, nil
}

func (d *dataAccess) Get() ([]byte, error) {
	return readAll(d.Reader())
}

func (d *dataAccess) Reader() (io.ReadCloser, error) {
	d.lock.Lock()
	reader := d.reader
	d.reader = nil
	d.lock.Unlock()
	if reader != nil {
		return reader, nil
	}
	return d.fetcher.Fetch(dummyContext, d.desc)
}

func readAll(reader io.ReadCloser, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func push(ctx context.Context, p resolve.Pusher, blob accessio.BlobAccess) error {
	desc := *artdesc.DefaultBlobDescriptor(blob)
	return pushData(ctx, p, desc, blob)
}

func pushData(ctx context.Context, p resolve.Pusher, desc artdesc.Descriptor, data accessio.DataAccess) error {
	key := remotes.MakeRefKey(ctx, desc)
	if desc.Size == 0 {
		desc.Size = -1
	}

	logging.Logger().Debug("*** push blob", "mediatype", desc.MediaType, "digest", desc.Digest, "key", key)
	req, err := p.Push(ctx, desc, data)
	if err != nil {
		if errdefs.IsAlreadyExists(err) {
			logging.Logger().Debug("blob already exists", "mediatype", desc.MediaType, "digest", desc.Digest)

			return nil
		}
		return fmt.Errorf("failed to push: %w", err)
	}
	return req.Commit(ctx, desc.Size, desc.Digest)
}

var dummyContext = nologger()

func nologger() context.Context {
	ctx := context.Background()
	logger := logrus.New()
	logger.Level = logrus.ErrorLevel
	return log.WithLogger(ctx, logrus.NewEntry(logger))
}
