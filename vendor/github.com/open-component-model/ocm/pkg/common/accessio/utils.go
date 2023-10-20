// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package accessio

import (
	"fmt"
	"io"

	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/common/compression"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

type closableReader struct {
	reader io.Reader
}

func ReadCloser(r io.Reader) io.ReadCloser { return closableReader{r} }

func (r closableReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r closableReader) Close() error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// NopWriteCloser returns a ReadCloser with a no-op Close method wrapping
// the provided Reader r.
func NopWriteCloser(w io.Writer) io.WriteCloser {
	return compression.NopWriteCloser(w)
}

////////////////////////////////////////////////////////////////////////////////

type additionalCloser struct {
	msg              []string
	reader           io.ReadCloser
	additionalCloser io.Closer
}

var _ io.ReadCloser = (*additionalCloser)(nil)

func AddCloser(reader io.ReadCloser, closer io.Closer, msg ...string) io.ReadCloser {
	return &additionalCloser{
		msg:              msg,
		reader:           reader,
		additionalCloser: closer,
	}
}

func (c *additionalCloser) Close() error {
	var list *errors.ErrorList
	if len(c.msg) == 0 {
		list = errors.ErrListf("close")
	} else {
		list = errors.ErrListf(c.msg[0], common.IterfaceSlice(c.msg[1:])...)
	}
	list.Add(c.reader.Close())
	list.Add(c.additionalCloser.Close())
	return list.Result()
}

func (c *additionalCloser) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

////////////////////////////////////////////////////////////////////////////////

func BlobData(blob DataGetter, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	return blob.Get()
}

func BlobReader(blob DataReader, err error) (io.ReadCloser, error) {
	if err != nil {
		return nil, err
	}
	return blob.Reader()
}

func FileSystem(fss ...vfs.FileSystem) vfs.FileSystem {
	return utils.FileSystem(fss...)
}

func DefaultedFileSystem(def vfs.FileSystem, fss ...vfs.FileSystem) vfs.FileSystem {
	return utils.DefaultedFileSystem(def, fss...)
}

type once struct {
	callbacks []CloserCallback
	closer    io.Closer
}

type CloserCallback func()

func OnceCloser(c io.Closer, callbacks ...CloserCallback) io.Closer {
	return &once{callbacks, c}
}

func (c *once) Close() error {
	if c.closer == nil {
		return nil
	}

	t := c.closer
	c.closer = nil
	err := t.Close()

	for _, cb := range c.callbacks {
		cb()
	}

	if err != nil {
		return fmt.Errorf("unable to close: %w", err)
	}

	return nil
}

func Close(closer ...io.Closer) error {
	if len(closer) == 0 {
		return nil
	}
	list := errors.ErrList()
	for _, c := range closer {
		if c != nil {
			list.Add(c.Close())
		}
	}
	return list.Result()
}

type Closer func() error

func (c Closer) Close() error {
	return c()
}
