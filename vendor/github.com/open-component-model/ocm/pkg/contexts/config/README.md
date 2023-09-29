# General Config Management for Contexts

The `config` context provides a generic configuration feature for data contexts

```go

type Context interface {
	datacontext.Context

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

	Generation() int64
	ApplyTo(gen int64, target interface{}) (int64, error)
}
```

It offers the possibility to register configuration objects of arbitrary types.
This is implemented by typed objects, like for the other kinds of data context.
Using this feature it is possible to decode configuration objects from
yaml documents or fragments.

A `Config` object must implement the `Config` interface:

```go
type Config interface {
	runtime.VersionedTypedObject

	ApplyTo(Context, interface{}) error
}
```

It can be called to be applied to a dedicated context (the second parameter)
under the control of a dedicated `Config` context (this first parameter).
If the object is not applicable for a given target context it should
ignore the call. So it is the decision of the config object, whether
it does something with the given context or not. It is just called for all
kinds of contexts requesting a replay of configuration operations regardless
of their type.

If applied using the such a configuration context by calling the appropriate
`Apply` method on  the context, the context keeps track of the sequence of
applied configuration  objects.

A configuration sink (any kind of data context) should keep track about
the latest applied configuration sequence number from its configuration context.
Whenever a non-trivial function is called on such a data context, it should request
a replay of missing configuration requests.

## Usage in Data Context

The update protocol for data contexts is supported by an `Updater` instance
that can be created for a data context.

It can be set by the context constructor:

```go
func newContext(shared datacontext.AttributesContext, configctx config.Context, reposcheme RepositoryTypeScheme, logger logging.Context) Context {
	c := &_context{
		sharedattributes:     shared,
		updater:              cfgcpi.NewUpdate(configctx),
		knownRepositoryTypes: reposcheme,
		...
	}
	c.Context = datacontext.NewContextBase(c, CONTEXT_TYPE, key, shared.GetAttributes(), logger)
	return c
}
```

Then the context implementation should provide an (internal) update function
called by methods based on configuration settings:

```go

func (c *_context) ConfigContext() config.Context {
	return c.updater.GetContext()
}

func (c *_context) Update() error {
	return c.updater.Update(c)
}
```

This will execute the sequence of missing configuration requests applied
to the configuration context used by this data context since the last update.

## Providing Configuration Objects

There might be any kind of configuration objects that can apply
themselves to any kind of data context.

It must implement the interface:

```go
type Config interface {
	runtime.VersionedTypedObject

	ApplyTo(Context, interface{}) error
}
```

Typically, such an object SHOULD provide a serialization format.
If supported it must get a unique type name, and it can be
registered for the configuration context.

The following snipped shows a typical pattern, how to implement this.

```go


const (
	MyConfigType   = "my.config" + common.TypeGroupSuffix
	MyConfigTypeV1 = MyConfigType + runtime.VersionSeparator + "v1"
)

func init() {
	config.RegisterConfigType(MyConfigType, config.NewConfigType(MyConfigType, &ConfigSpec{}))
	config.RegisterConfigType(MyConfigTypeV1, config.NewConfigType(MyConfigTypeV1, &ConfigSpec{}))
}

// ConfigSpec describes a memory based repository interface.
type ConfigSpec struct {
	runtime.ObjectVersionedType `json:",inline"`
	// Any configuration settings
	...
}

// NewConfigSpec creates a new memory ConfigSpec
func NewConfigSpec() *ConfigSpec {
	return &ConfigSpec{
		ObjectVersionedType: runtime.NewVersionedObjectType(MyConfigType),
	}
}

func (a *ConfigSpec) GetType() string {
	return MyConfigType
}

func (a *ConfigSpec) ApplyTo(ctx config.Context, target interface{}) error {
	// check if applicable for target object
	t, ok := target.(cpi.Context)
	if !ok {
		return config.ErrNoContext(MyConfigType)
	}
	// do the needful
	return false
}
```

If a data context provides an own type of configuration object this should
be implemented in sub package `config`.

## Generic Configuration

The configuration context provides an own configuration object for configuring
the configuration context (in sub package
`config`), that can be used to aggregate any kind of configuration object,
that is serializable.


```go
// Config describes a memory based repository interface.
type Config struct {
  runtime.ObjectVersionedType `json:",inline"`
  Configurations              []*cpi.GenericConfig `json:"configurations"`
}
```

This can be used to list any sequence of configurations that will be applied
in this order, if the object is applied to a configuration context.
The listed configuration requests will be stored, until they are replayed by a
dedicated data context.
