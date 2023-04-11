// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package datacontext

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/mandelsoft/logging"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext/action/handlers"
	"github.com/open-component-model/ocm/pkg/errors"
	ocmlog "github.com/open-component-model/ocm/pkg/logging"
	"github.com/open-component-model/ocm/pkg/runtime"
	"github.com/open-component-model/ocm/pkg/utils"
)

const OCM_CONTEXT_SUFFIX = ".context" + common.OCM_TYPE_GROUP_SUFFIX

// BuilderMode controls the handling of unset information in the
// builder configuration when calling the New method.
type BuilderMode int

const (
	// MODE_SHARED uses the default contexts for unset nested context types.
	MODE_SHARED BuilderMode = iota
	// MODE_DEFAULTED uses dedicated context instances configured with the
	// context type specific default registrations.
	MODE_DEFAULTED
	// MODE_EXTENDED uses dedicated context instances configured with
	// context type registrations extending the default registrations.
	MODE_EXTENDED
	// MODE_CONFIGURED uses dedicated context instances configured with the
	// context type registrations configured with the actual state of the
	// default registrations.
	MODE_CONFIGURED
	// MODE_INITIAL uses completely new contexts for unset nested context types
	// and initial registrations.
	MODE_INITIAL
)

func (m BuilderMode) String() string {
	switch m {
	case MODE_SHARED:
		return "shared"
	case MODE_DEFAULTED:
		return "defaulted"
	case MODE_EXTENDED:
		return "extended"
	case MODE_CONFIGURED:
		return "configured"
	case MODE_INITIAL:
		return "initial"
	default:
		return fmt.Sprintf("(invalid %d)", m)
	}
}

func Mode(m ...BuilderMode) BuilderMode {
	return utils.OptionalDefaulted(MODE_EXTENDED, m...)
}

type ContextProvider interface {
	// AttributesContext returns the shared attributes
	AttributesContext() AttributesContext
}

// Delegates is the interface for common
// Context features, which might be delegated
// to aggregated contexts.
type Delegates interface {
	ocmlog.LogProvider
	handlers.ActionsProvider
}

type _delegates struct {
	logging logging.Context
	actions handlers.Registry
}

func (d _delegates) LoggingContext() logging.Context {
	return d.logging
}

func (d _delegates) Logger(messageContext ...logging.MessageContext) logging.Logger {
	return d.logging.Logger(messageContext)
}

func (d _delegates) GetActions() handlers.Registry {
	return d.actions
}

func ComposeDelegates(l logging.Context, a handlers.Registry) Delegates {
	return _delegates{l, a}
}

// Context describes a common interface for a data context used for a dedicated
// purpose.
// Such has a type and always specific attribute store.
// Every Context can be bound to a context.Context.
type Context interface {
	ContextProvider
	Delegates

	// GetType returns the context type
	GetType() string

	// BindTo binds the context to a context.Context and makes it
	// retrievable by a ForContext method
	BindTo(ctx context.Context) context.Context
	GetAttributes() Attributes
}

////////////////////////////////////////////////////////////////////////////////

// CONTEXT_TYPE is the global type for an attribute context.
const CONTEXT_TYPE = "attributes" + OCM_CONTEXT_SUFFIX

type AttributesContext interface {
	Context

	BindTo(ctx context.Context) context.Context
}

// AttributeFactory is used to atomicly create a new attribute for a context.
type AttributeFactory func(Context) interface{}

type Attributes interface {
	GetAttribute(name string, def ...interface{}) interface{}
	SetAttribute(name string, value interface{}) error
	SetEncodedAttribute(name string, data []byte, unmarshaller runtime.Unmarshaler) error
	GetOrCreateAttribute(name string, creator AttributeFactory) interface{}
}

// DefaultContext is the default context initialized by init functions.
var DefaultContext = NewWithActions(nil, handlers.DefaultRegistry())

// ForContext returns the Context to use for context.Context.
// This is either an explicit context or the default context.
func ForContext(ctx context.Context) AttributesContext {
	c, _ := ForContextByKey(ctx, key, DefaultContext)
	if c == nil {
		return nil
	}
	return c.(AttributesContext)
}

// WithContext create a new Context bound to a context.Context.
func WithContext(ctx context.Context, parentAttrs Attributes) (Context, context.Context) {
	c := New(parentAttrs)
	return c, c.BindTo(ctx)
}

////////////////////////////////////////////////////////////////////////////////

type Updater interface {
	Update() error
}

type UpdateFunc func() error

func (u UpdateFunc) Update() error {
	return u()
}

type delegates = Delegates

type contextBase struct {
	ctxtype    string
	id         int64
	key        interface{}
	effective  Context
	attributes Attributes
	delegates
}

var _ Context = (*contextBase)(nil)

// NewContextBase creates a context base implementation supporting
// context attributes and the binding to a context.Context.
func NewContextBase(eff Context, typ string, key interface{}, parentAttrs Attributes, delegates Delegates) Context {
	updater, _ := eff.(Updater)
	c := &contextBase{ctxtype: typ, key: key, effective: eff}
	c.attributes = newAttributes(eff, parentAttrs, &updater)
	c.delegates = ComposeDelegates(logging.NewWithBase(delegates.LoggingContext()), handlers.NewRegistry(delegates.GetActions()))
	return c
}

func (c *contextBase) GetType() string {
	return c.ctxtype
}

// BindTo make the Context reachable via the resulting context.Context.
func (c *contextBase) BindTo(ctx context.Context) context.Context {
	return context.WithValue(ctx, c.key, c.effective)
}

func (c *contextBase) AttributesContext() AttributesContext {
	return c.effective.AttributesContext()
}

func (c *contextBase) GetAttributes() Attributes {
	return c.attributes
}

////////////////////////////////////////////////////////////////////////////////

type _context struct {
	Context
	updater Updater
}

var key = reflect.TypeOf(contextBase{})

// New provides a root attribute context.
func New(parentAttrs Attributes) AttributesContext {
	return NewWithActions(parentAttrs, handlers.NewRegistry(handlers.DefaultRegistry()))
}

func NewWithActions(parentAttrs Attributes, actions handlers.Registry) AttributesContext {
	c := &_context{}

	c.Context = &contextBase{
		ctxtype:    CONTEXT_TYPE,
		id:         contextrange.NextId(),
		key:        key,
		effective:  c,
		attributes: newAttributes(c, parentAttrs, &c.updater),
		delegates:  ComposeDelegates(logging.NewWithBase(ocmlog.Context()), handlers.NewRegistry(actions)),
	}
	return c
}

// AssureUpdater is used to assure the existence of an updater in
// a root context if a config context is down the context hierarchy.
// This method SHOULD only be called by a config context.
func AssureUpdater(attrs AttributesContext, u Updater) {
	c, ok := attrs.(*_context)
	if !ok {
		return
	}
	if c.updater == nil {
		c.updater = u
	}
}

func (c *_context) Actions() handlers.Registry {
	if c.updater != nil {
		c.updater.Update()
	}
	return c.Context.GetActions()
}

func (c *_context) LoggingContext() logging.Context {
	if c.updater != nil {
		c.updater.Update()
	}
	return c.Context.LoggingContext()
}

func (c *_context) Logger(messageContext ...logging.MessageContext) logging.Logger {
	if c.updater != nil {
		c.updater.Update()
	}
	return c.Context.Logger(messageContext...)
}

////////////////////////////////////////////////////////////////////////////////

// NumberRange can be used as source for successive id numbers to tag
// elements, since debuggers not always sow object addresses.
type NumberRange struct {
	id   int64
	lock sync.Mutex
}

func (n *NumberRange) NextId() int64 {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.id++
	return n.id
}

var contextrange, attrsrange = NumberRange{}, NumberRange{}

type _attributes struct {
	sync.RWMutex
	id         int64
	ctx        Context
	parent     Attributes
	updater    *Updater
	attributes map[string]interface{}
}

var _ Attributes = &_attributes{}

func NewAttributes(ctx Context, parent Attributes, updater *Updater) Attributes {
	return newAttributes(ctx, parent, updater)
}

func newAttributes(ctx Context, parent Attributes, updater *Updater) *_attributes {
	return &_attributes{
		id:         attrsrange.NextId(),
		ctx:        ctx,
		parent:     parent,
		updater:    updater,
		attributes: map[string]interface{}{},
	}
}

func (c *_attributes) GetAttribute(name string, def ...interface{}) interface{} {
	if *c.updater != nil {
		(*c.updater).Update()
	}
	c.RLock()
	defer c.RUnlock()
	if a := c.attributes[name]; a != nil {
		return a
	}
	if c.parent != nil {
		if a := c.parent.GetAttribute(name); a != nil {
			return a
		}
	}
	return utils.Optional(def...)
}

func (c *_attributes) SetEncodedAttribute(name string, data []byte, unmarshaller runtime.Unmarshaler) error {
	s := DefaultAttributeScheme.Shortcuts()[name]
	if s != "" {
		name = s
	}
	v, err := DefaultAttributeScheme.Decode(name, data, unmarshaller)
	if err != nil {
		return err
	}
	c.SetAttribute(name, v)
	return nil
}

func (c *_attributes) SetAttribute(name string, value interface{}) error {
	c.Lock()
	defer c.Unlock()

	_, err := DefaultAttributeScheme.Encode(name, value, nil)
	if err != nil && !errors.IsErrUnknownKind(err, "attribute") {
		return err
	}
	old := c.attributes[name]
	if old != nil && old != value {
		if c, ok := old.(io.Closer); ok {
			c.Close()
		}
	}
	c.attributes[name] = value
	return nil
}

func (c *_attributes) GetOrCreateAttribute(name string, creator AttributeFactory) interface{} {
	c.Lock()
	defer c.Unlock()
	if v := c.attributes[name]; v != nil {
		return v
	}
	if c.parent != nil {
		if v := c.parent.GetAttribute(name); v != nil {
			return v
		}
	}
	v := creator(c.ctx)
	c.attributes[name] = v
	return v
}
