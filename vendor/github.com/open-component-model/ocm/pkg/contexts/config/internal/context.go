// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"reflect"

	"github.com/mandelsoft/logging"

	"github.com/open-component-model/ocm/pkg/common"
	"github.com/open-component-model/ocm/pkg/contexts/datacontext"
	"github.com/open-component-model/ocm/pkg/errors"
	"github.com/open-component-model/ocm/pkg/runtime"
)

// OCM_CONFIG_TYPE_SUFFIX is the standard suffix used for configuration
// types provided by this library.
const OCM_CONFIG_TYPE_SUFFIX = ".config" + common.OCM_TYPE_GROUP_SUFFIX

type ConfigSelector interface {
	Select(Config) bool
}
type ConfigSelectorFunction func(Config) bool

func (f ConfigSelectorFunction) Select(cfg Config) bool { return f(cfg) }

var AllConfigs = AppliedConfigSelectorFunction(func(*AppliedConfig) bool { return true })

const AllGenerations int64 = 0

const CONTEXT_TYPE = "config" + datacontext.OCM_CONTEXT_SUFFIX

type ContextProvider interface {
	ConfigContext() Context
}

type Context interface {
	datacontext.Context
	ContextProvider

	AttributesContext() datacontext.AttributesContext

	// Info provides the context for nested configuration evaluation
	Info() string
	// WithInfo provides the same context with additional nesting info
	WithInfo(desc string) Context

	ConfigTypes() ConfigTypeScheme

	// GetConfigForData deserialize configuration objects for known
	// configuration types.
	GetConfigForData(data []byte, unmarshaler runtime.Unmarshaler) (Config, error)

	// ApplyData applies the config given by a byte stream to the config store
	// If the config type is not known, a generic config is stored and returned.
	// In this case an unknown error for kind KIND_CONFIGTYPE is returned.
	ApplyData(data []byte, unmarshaler runtime.Unmarshaler, desc string) (Config, error)
	// ApplyConfig applies the config to the config store
	ApplyConfig(spec Config, desc string) error

	GetConfigForType(generation int64, typ string) (int64, []Config)
	GetConfigForName(generation int64, name string) (int64, []Config)
	GetConfig(generation int64, selector ConfigSelector) (int64, []Config)

	// Reset all configs applied so far, subsequent calls to ApplyTo will
	// ony see configs allpied after the last reset.
	Reset() int64
	// Generation return the actual config generation.
	// this is a strictly increasing number, regardless of the number
	// of Reset calls.
	Generation() int64
	// ApplyTo applies all configurations applied after the last reset with
	// a generation larger than the given watermark to the specified target.
	// A target may be any object. The applied configuration objects decide
	// on their own whether they are applicable for the given target.
	// The generation of the last applied object is returned to be used as
	// new watermark.
	ApplyTo(gen int64, target interface{}) (int64, error)
}

var key = reflect.TypeOf(_context{})

// DefaultContext is the default context initialized by init functions.
var DefaultContext = Builder{}.New(datacontext.MODE_SHARED)

// ForContext returns the Context to use for context.Context.
// This is either an explicit context or the default context.
// The returned context incorporates the given context.
func ForContext(ctx context.Context) Context {
	c, _ := datacontext.ForContextByKey(ctx, key, DefaultContext)
	return c.(Context)
}

func DefinedForContext(ctx context.Context) (Context, bool) {
	c, ok := datacontext.ForContextByKey(ctx, key, DefaultContext)
	if c != nil {
		return c.(Context), ok
	}
	return nil, ok
}

////////////////////////////////////////////////////////////////////////////////

type coreContext struct {
	datacontext.Context
	updater Updater

	sharedAttributes datacontext.AttributesContext

	knownConfigTypes ConfigTypeScheme

	configs *ConfigStore
}

type _context struct {
	*coreContext
	description string
}

var _ Context = &_context{}

func newContext(shared datacontext.AttributesContext, reposcheme ConfigTypeScheme, logger logging.Context) Context {
	c := &_context{
		coreContext: &coreContext{
			sharedAttributes: shared,
			knownConfigTypes: reposcheme,
			configs:          NewConfigStore(),
		},
	}
	c.Context = datacontext.NewContextBase(c, CONTEXT_TYPE, key, shared.GetAttributes(), logger)
	c.updater = NewUpdater(c, c)
	datacontext.AssureUpdater(shared, NewUpdater(c, shared))
	return c
}

func (c *_context) ConfigContext() Context {
	return c
}

func (c *_context) Update() error {
	return c.updater.Update()
}

var _ datacontext.Updater = (*_context)(nil)

func (c *_context) Info() string {
	return c.description
}

func (c *_context) WithInfo(desc string) Context {
	if c.description != "" {
		desc = desc + "--" + c.description
	}
	return &_context{c.coreContext, desc}
}

func (c *_context) AttributesContext() datacontext.AttributesContext {
	c.updater.Update()
	return c.sharedAttributes
}

func (c *_context) ConfigTypes() ConfigTypeScheme {
	return c.knownConfigTypes
}

func (c *_context) ConfigForData(data []byte, unmarshaler runtime.Unmarshaler) (Config, error) {
	return c.knownConfigTypes.DecodeConfig(data, unmarshaler)
}

func (c *_context) GetConfigForData(data []byte, unmarshaler runtime.Unmarshaler) (Config, error) {
	spec, err := c.knownConfigTypes.DecodeConfig(data, unmarshaler)
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func (c *_context) ApplyConfig(spec Config, desc string) error {
	var unknown error
	spec = (&AppliedConfig{config: spec}).eval(c)
	if IsGeneric(spec) {
		unknown = errors.ErrUnknown(KIND_CONFIGTYPE, spec.GetType())
	}

	ctx := c.WithInfo(desc)
	err := spec.ApplyTo(c, ctx)
	if IsErrNoContext(err) {
		err = unknown
	}
	err = errors.Wrapf(err, ctx.Info())
	c.configs.Apply(spec, ctx.Info())
	return err
}

func (c *_context) ApplyData(data []byte, unmarshaler runtime.Unmarshaler, desc string) (Config, error) {
	spec, err := c.knownConfigTypes.DecodeConfig(data, unmarshaler)
	if err != nil {
		return nil, err
	}
	return spec, c.ApplyConfig(spec, desc)
}

func (c *_context) selector(gen int64, selector ConfigSelector) AppliedConfigSelector {
	if gen <= 0 {
		return AppliedConfigSelectorFor(selector)
	}
	if selector == nil {
		return AppliedGenerationSelector(gen)
	}
	return AppliedAndSelector(AppliedGenerationSelector(gen), AppliedConfigSelectorFor(selector))
}

func (c *_context) Generation() int64 {
	return c.configs.Generation()
}

func (c *_context) Reset() int64 {
	return c.configs.Reset()
}

func (c *_context) ApplyTo(gen int64, target interface{}) (int64, error) {
	cur := c.configs.Generation()
	if cur <= gen {
		return gen, nil
	}
	cur, cfgs := c.configs.GetConfigForSelector(c, AppliedGenerationSelector(gen))

	list := errors.ErrListf("config apply errors")
	for _, cfg := range cfgs {
		err := errors.Wrapf(cfg.config.ApplyTo(c, target), cfg.description)
		if !IsErrNoContext(err) {
			list.Add(err)
		}
	}
	return cur, list.Result()
}

func (c *_context) GetConfig(gen int64, selector ConfigSelector) (int64, []Config) {
	gen, cfgs := c.configs.GetConfigForSelector(c, c.selector(gen, selector))
	return gen, cfgs.Configs()
}

func (c *_context) GetConfigForName(gen int64, name string) (int64, []Config) {
	gen, cfgs := c.configs.GetConfigForName(c, name, c.selector(gen, nil))
	return gen, cfgs.Configs()
}

func (c *_context) GetConfigForType(gen int64, typ string) (int64, []Config) {
	gen, cfgs := c.configs.GetConfigForType(c, typ, c.selector(gen, nil))
	return gen, cfgs.Configs()
}
