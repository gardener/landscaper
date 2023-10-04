// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"io"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
)

type ReaderProvider interface {
	Reader() (io.ReadCloser, error)
}

type OnDemandReader struct {
	lock     sync.Mutex
	provider ReaderProvider
	reader   io.ReadCloser
	err      error
}

var _ io.Reader = (*OnDemandReader)(nil)

func NewOndemandReader(p ReaderProvider) io.ReadCloser {
	return &OnDemandReader{provider: p}
}

func (o *OnDemandReader) Read(p []byte) (n int, err error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if o.reader == nil {
		r, err := o.provider.Reader()
		if err != nil {
			o.err = err
			return 0, err
		}
		o.reader = r
	}
	data, err := o.reader.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		o.err = err
	}
	return data, err
}

func (o *OnDemandReader) Close() error {
	o.lock.Lock()
	defer o.lock.Unlock()

	if o.reader == nil {
		return o.err
	}
	if o.err != nil {
		o.reader.Close()
		return o.err
	}
	return o.reader.Close()
}
