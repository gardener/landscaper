// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"bytes"

	"github.com/mandelsoft/logging"
	"github.com/tonglil/buflogr"
)

type LogProvider interface {
	logging.ContextProvider
	Logger(messageContext ...logging.MessageContext) logging.Logger
}

func NewDefaultContext() logging.Context {
	return NewContext(logging.DefaultContext())
}

func NewBufferedContext() (logging.Context, *bytes.Buffer) {
	var buf bytes.Buffer
	return logging.New(buflogr.NewWithBuffer(&buf)), &buf
}
