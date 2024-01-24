// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"io"

	"github.com/opencontainers/go-digest"

	"github.com/open-component-model/ocm/pkg/blobaccess"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/iotools"
)

type Writer interface {
	io.Writer
	io.WriterAt
}

type DataWriter interface {
	WriteTo(Writer) (int64, digest.Digest, error)
}

////////////////////////////////////////////////////////////////////////////////

type readerWriter struct {
	reader io.ReadCloser
}

func NewReaderWriter(r io.ReadCloser) DataWriter {
	return &readerWriter{r}
}

func (d *readerWriter) WriteTo(w Writer) (size int64, dig digest.Digest, err error) {
	defer errors.PropagateError(&err, d.reader.Close)
	dr := iotools.NewDefaultDigestReader(d.reader)
	_, err = io.Copy(w, dr)
	if err != nil {
		return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, err
	}
	return dr.Size(), dr.Digest(), err
}

type dataAccessWriter struct {
	access blobaccess.DataAccess
}

func NewDataAccessWriter(acc blobaccess.DataAccess) DataWriter {
	return &dataAccessWriter{acc}
}

func (d *dataAccessWriter) WriteTo(w Writer) (int64, digest.Digest, error) {
	r, err := d.access.Reader()
	if err != nil {
		return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, err
	}
	return (&readerWriter{r}).WriteTo(w)
}

type writerAtWrapper struct {
	writer func(w io.WriterAt) error
}

func NewWriteAtWriter(at func(w io.WriterAt) error) DataWriter {
	return &writerAtWrapper{at}
}

func (d *writerAtWrapper) WriteTo(w Writer) (int64, digest.Digest, error) {
	return BLOB_UNKNOWN_SIZE, BLOB_UNKNOWN_DIGEST, d.writer(w)
}
