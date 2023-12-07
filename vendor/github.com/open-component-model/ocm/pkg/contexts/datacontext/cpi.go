// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package datacontext

import (
	"context"
	"fmt"
	runtime2 "runtime"

	"github.com/mandelsoft/logging"

	"github.com/open-component-model/ocm/pkg/contexts/datacontext/action/handlers"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/finalizer"
)

// NewContextBase creates a context base implementation supporting
// context attributes and the binding to a context.Context.
func NewContextBase(eff Context, typ string, key interface{}, parentAttrs Attributes, delegates Delegates) InternalContext {
	updater, _ := eff.(Updater)
	return newContextBase(eff, typ, key, parentAttrs, &updater,
		ComposeDelegates(logging.NewWithBase(delegates.LoggingContext()), handlers.NewRegistry(nil, delegates.GetActions())),
	)
}

// GCWrapper is the embeddable base type
// for a context wrapper handling garbage collection.
// It aölso handles the BindTo interface for a context.
type GCWrapper struct {
	self Context
	key  interface{}
}

// setSelf is not public to enforce
// the usage of this GCWrapper type in context
// specific garbage collection wrappers.
// It is enforced by the
// finalizableContextWrapper interface.
func (w *GCWrapper) setSelf(self Context, key interface{}) {
	w.self = self
	w.key = key
}

func init() { // linter complains about unused method.
	(&GCWrapper{}).setSelf(nil, nil)
}

// BindTo makes the Context reachable via the resulting context.Context.
// Go requires not to use a pointer receiver, here ??????
func (b GCWrapper) BindTo(ctx context.Context) context.Context {
	return context.WithValue(ctx, b.key, b.self)
}

// finalizableContextWrapper is the interface for
// a context wrapper used to establish a garbage collectable
// runtime finalizer.
// It is a helper interface for Go generics to enforce a
// struct pointer.
type finalizableContextWrapper[C InternalContext, P any] interface {
	InternalContext

	SetContext(C)
	setSelf(Context, interface{})
	*P
}

// FinalizedContext wraps a context implementation C into a separate wrapper
// object of type *W and returns this wrapper.
// It should have the type
//
//	struct {
//	   C
//	}
//
// The wrapper is created and a runtime finalizer is
// defined for this object, which calls the Finalize Method on the
// context implementation.
func FinalizedContext[W Context, C InternalContext, P finalizableContextWrapper[C, W]](c C) P {
	var v W
	p := (P)(&v)
	p.SetContext(c)
	p.setSelf(p, c.GetKey()) // prepare for generic bind operation
	runtime2.SetFinalizer(p, fi[W, C, P])
	Debug(p, "create context", "id", c.GetId())
	return p
}

func fi[W Context, C InternalContext, P finalizableContextWrapper[C, W]](c P) {
	err := c.Cleanup()
	c.GetRecorder().Record(c.GetId())
	Debug(c, "cleanup context", "error", err)
}

type contextBase struct {
	ctxtype    string
	id         ContextIdentity
	key        interface{}
	effective  Context
	attributes *_attributes
	delegates

	finalizer *finalizer.Finalizer
	recorder  *finalizer.RuntimeFinalizationRecoder
}

var _ Context = (*contextBase)(nil)

func newContextBase(eff Context, typ string, key interface{}, parentAttrs Attributes, updater *Updater, delegates Delegates) *contextBase {
	recorder := &finalizer.RuntimeFinalizationRecoder{}
	id := ContextIdentity(fmt.Sprintf("%s/%d", typ, contextrange.NextId()))
	c := &contextBase{
		ctxtype:    typ,
		id:         id,
		key:        key,
		effective:  eff,
		finalizer:  &finalizer.Finalizer{},
		attributes: newAttributes(eff, parentAttrs, updater),
		delegates:  delegates,
		recorder:   recorder,
	}
	return c
}

func (c *contextBase) BindTo(ctx context.Context) context.Context {
	panic("should never be called")
}

func (c *contextBase) GetType() string {
	return c.ctxtype
}

func (c *contextBase) GetId() ContextIdentity {
	return c.id
}

func (c *contextBase) GetKey() interface{} {
	return c.key
}

func (c *contextBase) AttributesContext() AttributesContext {
	return c.effective.AttributesContext()
}

func (c *contextBase) GetAttributes() Attributes {
	return c.attributes
}

func (c *contextBase) GetRecorder() *finalizer.RuntimeFinalizationRecoder {
	return c.recorder
}

func (c *contextBase) Cleanup() error {
	list := errors.ErrListf("cleanup %s", c.id)
	list.Addf(nil, c.finalizer.Finalize(), "finalizers")
	list.Add(c.attributes.Finalize())
	return list.Result()
}

func (c *contextBase) Finalize() error {
	return c.finalizer.Finalize()
}

func (c *contextBase) Finalizer() *finalizer.Finalizer {
	return c.finalizer
}

// AssureUpdater is used to assure the existence of an updater in
// a root context if a config context is down the context hierarchy.
// This method SHOULD only be called by a config context.
func AssureUpdater(attrs AttributesContext, u Updater) {
	c, ok := attrs.(*gcWrapper)
	if !ok {
		return
	}
	if c.updater == nil {
		c.updater = u
	}
}
