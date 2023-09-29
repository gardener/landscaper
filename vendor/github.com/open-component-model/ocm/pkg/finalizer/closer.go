// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package finalizer

import (
	"io"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/errors"
)

type readcloser = io.ReadCloser

type finalizingCloser struct {
	readcloser
	msg       []string
	finalizer *Finalizer
}

var _ io.ReadCloser = (*finalizingCloser)(nil)

func addToCloser(reader io.ReadCloser, f *Finalizer, msg ...string) io.ReadCloser {
	return &finalizingCloser{
		readcloser: reader,
		msg:        msg,
		finalizer:  f,
	}
}

func (c *finalizingCloser) Close() error {
	var list *errors.ErrorList
	if len(c.msg) == 0 {
		list = errors.ErrListf("close")
	} else {
		list = errors.ErrListf(c.msg[0], common.IterfaceSlice(c.msg[1:])...)
	}
	list.Add(c.readcloser.Close())
	if c.finalizer != nil {
		list.Add(c.finalizer.Finalize())
	}
	return list.Result()
}
