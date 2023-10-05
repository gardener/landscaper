// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"bytes"
	"io"
	"os"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/open-component-model/ocm/pkg/errors"
)

type ResettableReader struct {
	orig  io.ReadCloser
	buf   Buffer
	count int
}

func NewResettableReader(orig io.ReadCloser, size int64) (*ResettableReader, error) {
	var buf Buffer
	var err error

	if size < 0 || size > 8192 {
		buf, err = NewFileBuffer()
		if err != nil {
			return nil, err
		}
	} else {
		buf = &memoryBuffer{}
	}
	return &ResettableReader{
		orig: orig, buf: buf,
	}, nil
}

func (b *ResettableReader) Read(out []byte) (int, error) {
	n, err := b.orig.Read(out)
	if n > 0 {
		return b.buf.Write(out[:n])
	}
	return n, err
}

func (b *ResettableReader) Close() error {
	logrus.Debugf("close resend buffer\n")
	b.buf.Close()
	b.buf = nil
	return b.orig.Close()
}

func (b *ResettableReader) Reset() (io.ReadCloser, error) {
	b.count++
	if b.buf.Len() <= 0 {
		return &prefixReader{
			nil,
			b,
		}, nil
	}
	r, err := b.buf.Reader()
	if err != nil {
		return nil, err
	}
	return &prefixReader{
		r,
		b,
	}, nil
}

type prefixReader struct {
	prefix io.ReadCloser
	resend *ResettableReader
}

func (p *prefixReader) Read(out []byte) (int, error) {
	if p.prefix != nil {
		n, err := p.prefix.Read(out)
		if err == nil {
			return n, nil
		}
		p.prefix.Close()
		p.prefix = nil
	}
	n, err := p.resend.Read(out)
	logrus.Debugf("blob read %d: %s\n", n, err)
	return n, err
}

func (p *prefixReader) Close() error {
	logrus.Debugf("close prefix reader\n")
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type Buffer interface {
	Write(out []byte) (int, error)
	Reader() (io.ReadCloser, error)
	Len() int
	Close() error
	Release() error
}

type memoryBuffer struct {
	bytes.Buffer
}

var _ Buffer = (*memoryBuffer)(nil)

func (m *memoryBuffer) Reader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.Bytes())), nil
}

func (m *memoryBuffer) Close() error {
	return nil
}

func (m *memoryBuffer) Release() error {
	return nil
}

type fileBuffer struct {
	lock      sync.RWMutex
	readcount int
	path      string
	file      *os.File
}

var _ Buffer = (*fileBuffer)(nil)

func NewFileBuffer() (*fileBuffer, error) {
	file, err := os.CreateTemp("", "ociblob*")
	if err != nil {
		return nil, err
	}
	return &fileBuffer{
		path: file.Name(),
		file: file,
	}, nil
}

func (b *fileBuffer) Write(out []byte) (int, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.file == nil {
		return 0, ErrClosed
	}
	return b.file.Write(out)
}

func (b *fileBuffer) Reader() (io.ReadCloser, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.file == nil {
		return nil, ErrClosed
	}
	r, err := os.Open(b.path)
	if err != nil {
		return nil, err
	}
	b.readcount++
	return &bufferReader{buffer: b, ReadCloser: r}, nil
}

func (b *fileBuffer) Len() int {
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.file == nil {
		return -1
	}

	fi, err := b.file.Stat()
	if err != nil {
		return -1
	}
	return int(fi.Size())
}

func (b *fileBuffer) Close() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.file == nil {
		return ErrClosed
	}
	return b.file.Close()
}

func (b *fileBuffer) Release() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.file == nil {
		return nil
	}
	// just assure file to be closed
	_ = b.file.Close()
	b.file = nil
	if b.readcount == 0 {
		return os.Remove(b.path)
	}
	return nil
}

type bufferReader struct {
	io.ReadCloser
	buffer *fileBuffer
}

func (b *bufferReader) Close() error {
	b.buffer.lock.Lock()
	defer b.buffer.lock.Unlock()

	if b.ReadCloser == nil {
		return ErrClosed
	}
	list := errors.ErrListf("closing file buffer")
	r := b.ReadCloser
	b.ReadCloser = nil
	b.buffer.readcount--
	list.Add(r.Close())
	if b.buffer.readcount <= 0 && b.buffer.file == nil {
		list.Add(os.Remove(b.buffer.path))
	}
	return list.Result()
}
