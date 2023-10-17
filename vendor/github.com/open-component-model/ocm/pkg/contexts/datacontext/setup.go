// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package datacontext

import (
	"sync"
)

// SetupHandler is a handler, which can be registered by an arbitrary
// context part used to setup this parts for a new context.
// For example, a typical use case is setup a default value
// for a context attribute.
// The handler may consider the context creation mode to determine
// the initial content or structure of the context part.
type SetupHandler interface {
	Setup(mode BuilderMode, ctx Context)
}

// SetupHandlerFunction is a function usable as SetupHandler.
type SetupHandlerFunction func(mode BuilderMode, ctx Context)

func (f SetupHandlerFunction) Setup(mode BuilderMode, ctx Context) {
	f(mode, ctx)
}

// SetupHandlerRegistry is used to register and execute SetupHandler for
// a freshly created context.
type SetupHandlerRegistry interface {
	Register(h SetupHandler)
	Setup(mode BuilderMode, ctx Context)
}

type setupRegistry struct {
	lock     sync.Mutex
	handlers []SetupHandler
}

func (s *setupRegistry) Register(h SetupHandler) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.handlers = append(s.handlers, h)
}

func (s *setupRegistry) Setup(mode BuilderMode, ctx Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, h := range s.handlers {
		h.Setup(mode, ctx)
	}
}

var registry SetupHandlerRegistry = &setupRegistry{}

func RegisterSetupHandler(h SetupHandler) {
	registry.Register(h)
}

func SetupContext[C Context](mode BuilderMode, ctx C) C {
	registry.Setup(mode, ctx)
	return ctx
}
