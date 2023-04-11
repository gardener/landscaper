// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package registrations provides a hierarchical namespace for
// denoting any kind of handlers to be registered on some target.
// Handlers are denoted by names evaluated by HandlerRegistrationHandler
// Such a registration handler is responsible vor a complete sub namespace
// and may delegate the evaluation to nested handler mounted on a sub namespace.
package registrations

import (
	"fmt"
	"strings"
	"sync"

	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/utils"
)

type HandlerConfig interface{}

type HandlerRegistrationHandler[T any, O any] interface {
	RegisterByName(handler string, target T, config HandlerConfig, opts ...O) (bool, error)
}

type HandlerRegistrationRegistry[T any, O any] interface {
	HandlerRegistrationHandler[T, O]
	RegisterRegistrationHandler(path string, handler HandlerRegistrationHandler[T, O])
	GetRegistrationHandlers(name string) []*RegistrationHandlerInfo[T, O]
}

type NamePath []string

func NewNamePath(path string) NamePath {
	return strings.Split(path, "/")
}

func (p NamePath) Compare(o NamePath) int {
	if d := len(p) - len(o); d != 0 {
		return d
	}
	for i, e := range p {
		if d := strings.Compare(e, o[i]); d != 0 {
			return d
		}
	}
	return 0
}

func (p NamePath) IsPrefixOf(o NamePath) bool {
	if len(p) > len(o) {
		return false
	}
	for i, e := range p {
		if e != o[i] {
			return false
		}
	}
	return true
}

type RegistrationHandlerInfo[T any, O any] struct {
	prefix  NamePath
	handler HandlerRegistrationHandler[T, O]
}

func NewRegistrationHandlerInfo[T any, O any](path string, handler HandlerRegistrationHandler[T, O]) *RegistrationHandlerInfo[T, O] {
	return &RegistrationHandlerInfo[T, O]{
		prefix:  NewNamePath(path),
		handler: handler,
	}
}

func (i *RegistrationHandlerInfo[T, O]) RegisterByName(handler string, target T, config HandlerConfig, opts ...O) (bool, error) {
	path := NewNamePath(handler)

	if !i.prefix.IsPrefixOf(path) {
		return false, nil
	}
	return i.handler.RegisterByName(strings.Join(path[len(i.prefix):], "/"), target, config, opts...)
}

type handlerRegistrationRegistry[T any, O any] struct {
	lock     sync.RWMutex
	base     HandlerRegistrationRegistry[T, O]
	handlers []*RegistrationHandlerInfo[T, O]
}

func NewHandlerRegistrationRegistry[T any, O any](base ...HandlerRegistrationRegistry[T, O]) HandlerRegistrationRegistry[T, O] {
	return &handlerRegistrationRegistry[T, O]{base: utils.Optional(base...)}
}

func (c *handlerRegistrationRegistry[T, O]) RegisterRegistrationHandler(path string, handler HandlerRegistrationHandler[T, O]) {
	c.lock.Lock()
	defer c.lock.Unlock()

	comps := strings.Split(path, "/")
	n := &RegistrationHandlerInfo[T, O]{
		prefix:  comps,
		handler: handler,
	}

	var i int
	var h *RegistrationHandlerInfo[T, O]
	for i, h = range c.handlers {
		if h.prefix.Compare(comps) < 0 {
			break
		}
	}
	c.handlers = append(c.handlers[:i], append([]*RegistrationHandlerInfo[T, O]{n}, c.handlers[i:]...)...)
}

func (c *handlerRegistrationRegistry[T, O]) GetRegistrationHandlers(name string) []*RegistrationHandlerInfo[T, O] {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var result []*RegistrationHandlerInfo[T, O]
	path := NewNamePath(name)
	for _, h := range c.handlers {
		if h.prefix.IsPrefixOf(path) {
			result = append(result, h)
		}
	}

	if c.base != nil {
		base := c.base.GetRegistrationHandlers(name)
		i := 0
		for _, h := range base {
			for i != len(result) && result[i].prefix.Compare(h.prefix) >= 0 {
				i++
			}
			result = append(result[:i], append([]*RegistrationHandlerInfo[T, O]{h}, result[i:]...)...)
			i++
		}
	}
	return result
}

func (c *handlerRegistrationRegistry[T, O]) RegisterByName(handler string, target T, config HandlerConfig, opts ...O) (bool, error) {
	list := c.GetRegistrationHandlers(handler)
	errlist := errors.ErrListf("handler registration")
	for _, h := range list {
		ok, err := h.RegisterByName(handler, target, config, opts...)
		if ok {
			return ok, err
		}
		errlist.Add(err)
	}
	if errlist.Len() > 0 {
		return false, errlist.Result()
	}
	return false, fmt.Errorf("no registration handler found for %s", handler)
}
