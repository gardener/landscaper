# Logging for Go with context-specific Log Level Settings

This package provides a wrapper around the [logr](https://github.com/go-logr/logr)
logging system supporting a rule based approach to enable log levels
for dedicated message contexts specified at the logging location.

The rule set is configured for a logging context:

```go
    ctx := logging.New(logrLogger)
```

Any `logr.Logger` can be passed here, the level for this logger
is used as base level for the `ErrorLevel` of loggers provided
by the logging context.
If the full control should be handed over to the logging context, 
the maximum log level should be used for the sink of this logger.

If the used base level should always be 0, the base logger has to 
be set with plain mode:

```go
    ctx.SetBaseLogger(logrLogger, true)
```

Now you can add rules controlling the accepted log levels for dedicated log 
locations. First, a default log level can be set:

```go
    ctx.SetDefaultLevel(logging.InfoLevel)
```

This level restriction is used, if no rule matches a dedicated log request.

Another way to achieve the same goal is to provide a generic level rule without any
condition:

```go
    ctx.AddRule(logging.NewConditionRule(logging.InfoLevel))
```

A first rule for influencing the log level could be a realm rule.
A *Realm* represents a dedicated logical area, a good practice could be 
to use package names as realms. Realms are hierarchical consisting of
name components separated by a slash (/).

```go
    ctx.AddRule(logging.NewConditionRule(logging.DebugLevel, logging.NewRealm("github.com/mandelsoft/spiff")))
```

Alternatively `NewRealmPrefix(...)` can be used to match a complete realm hierarchy.

A realm for the actual package can be defined as local variable by using the
`Package` function:

```go
var realm = logging.Package()
```

Instead of passing `Logger`s around, now the logging `Context` is used.
It provides a method to access a logger specific for a dedicated log
request, for example, for a dedicated realm.

```go
  ctx.Logger(realm).Info("my message")
```

The provided logger offers the level specific functions, `Error`, `Warn`, `Info`, `Debug` and `Trace`.
Depending on the rule set configured for the used logging context, the level
for the given message context decides, which message to pass to the log sink of
the initial `logr.Logger`.

Alternatively a traditional `logr.Logger` for the given message context can be
obtained by using the `V` method:

```go
  ctx.V(logging.InfoLevel, realm).Info("my message")
```

The sink for this logger is configured to accept messages according to the
log level determined by th rule set of the logging context for the given
message context.

*Remark*: Returned `logr.Logger`s are always using a sink with the base level 0,
which is potentially shifted to the level of the base `logr.Logger`
used to set up the context, when forwarding to the original sink. This means
they are always directly using the log levels 0..*n*.

It is possible to get a loggings context with a predefined message context
with

```go
  ctx.WithContext("my message")
```

All loggers obtained from such a context will implicitly use the given
message context.

If no rules are configured, the default logger of the context is used
independently of the  given arguments. The given message context information is
optionally passed to the provided logger, depending on the used 
message context type.

For example, the realm is added to the logger's name.

It is also possible to provide dedicated attributes for the rule matching
process:

```go
  ctx.Logger(realm, logging.NewAttribute("test", "value")).Info("my message")
```

Such an attribute can be used as rule condition, also. This way, logging
can be enabled, for dedicated argument values of a method/function.

Both sides, the rule conditions and the message context can be a list.
For the conditions, all specified conditions must be evaluated to true, to
enable the rule. A rule is evaluated against the complete message context of
the log requests.
The default `ConditionRule` evaluates the rules against the complete log
request and a condition is *true*, if it matches at least one argument.

The rules are evaluated in the reverse order of their definition.
The first matching rule defines the finally used log level restriction and log
sink.

A `Rule` has the complete control over composing an appropriate logger.
The default condition based rule just enables the specified log level,
if all conditions match the actual log request.

For more complex conditions it is possible to compose conditions
using an `Or`, `And`, or `Not` condition.

Because `Rule` and `Condition` are interfaces, any desired behaviour
can be provided by dedicated rule and/or condition implementations.

## Default Logging Environment

This logging library provides a default logging context, it can be obtained
by

```go
  ctx := logging.DefaultContext()
```

This way it can be configured, also. It can be used for logging requests
not related to a dedicated logging context.

There is a shortcut to provide a logger for a message context based on
this default context:

```go
  logging.Log(messageContext).Debug(...)
```

or

```go
  logging.Log().V(logging.DebugLevel).Info(...
```

## Configuration

It is possible to configure a logging context from a textual configuration
using `config.ConfigureWithData(ctx, bytedata)`:

```yaml
defaultLevel: Info
rules:
  - rule:
      level: Debug
      conditions:
        - realm: github.com/mandelsoft/spiff
  - rule:
      level: Trace
      conditions:
        - attribute:
            name: test
            value:
               value: testvalue  # value is the *value* type, here
```

Rules might provide a deserialization by registering a type object
with `config.RegisterRuleType(name, typ)`. The factory type must implement the
interface `scheme.RuleType` and provide a value object
deserializable by yaml.

In a similar way it is possible to register a deserialization for
`Condition`s. The standard condition rule supports a condition deserialization
based on those registrations.

The standard names for rules are:
 - `rule`: condition rule

The standard names for conditions are:
- `and`: AND expression for a list of sub sequent conditions
- `or`: OR expression for a list of sub sequent conditions
- `not`: negate given expression
- `realm`: name for a realm condition
- `realmprefix`: name for a realm prefix condition
- `attribute`: attribute condition given by a map with `name` and `value`.
  
The config package also offers a value deserialization using
`config.RegisterValueType`. The default value type is `value`. 
It supports an `interface{}` deserialization.

For all deserialization types flat names are reserved for
the global usage by this library. Own types should use a reverse
DNS name to avoid conflicts by different users of this logging
API.

To provide own deserialization context, an own object of type
`config.Registry` can be created using `config.NewRegistry`.
The standard registry can be obtained by `config.DefaultRegistry()`

## Nesting Contexts

Logging contents can inherit from base contexts. This way the rule set,
logger and default level settings can be reused for a sub-level context.
THis contexts then provides a new scope to define additional rules
and settings only valid for this nested context. Settings done here are not
visible to log requests evaluated against the base context.

If a nested context defines an own base logger, the rules inherited from the base
context are evaluated against this logger if evaluated for a message
context passed to the nested context (extended-self principle).

A logging context reusing the settings provided by the default logging
context can be obtained by:

```go
  ctx := logging.NewWithBase(logging.DefaultContext())
```

## Preconfigured Rules, Message Contexts and Conditions

### Rules

The base library provides the following basic rule implementations.
It is possible to define own more complex rules by implementing
the `logging.Rule` interface.

- `NewRule(level, conditions...)` a simple rule setting a log level
for a message context matching all given conditions.

### Message Contexts and Conditions

The message context is a set of objects describing the context of a
log message. It can be used
- to enrich the log message
- ro enrich the logger (logr.Logger features a name to represent
  the call hierarchy when passing loggers to functions)
- to control the effective log condition based of configuration rules.
  (for example to enable all Info logs for log requests with a dedicated attribute)
 
The base library already provides some ready to use conditions
and message contexts:

- `Name`(*string*)  is attached as additional name part to the logr.Logger. 
  It cannot be used to control the log state.,

- `Tag`(*string*) Just some tag for a log request.
  Used as message context, the tag name is not added to the logger name for
  the log request.

- `Realm`(*string*) the location context of a logging request. This could
  be some kind of denotation for a functional area or Go package. To obtain the
  package realm for some coding the function `logging.Package()` can be used. Used as message context, the realm name is added as additional attribute (`realm`) to log message. As condition realms only match the last realm in a message context.

- `RealmPrefix`(*string*) (only as condition) matches against a complete 
  realm tree specified by a base realm. It matches the last realm in a message
  context, only.

- `Attribute`(*string,interface{}*) the name of an arbitrary attribute with some
  value. Used as message context, the key/value pair is added to the log message.

Meaning of predefined objects in a message context:

| Element     |  Rule Condition  | Message Context | Logger  |  LogMessage Attribute  |
|-------------|:----------------:|:---------------:|:-------:|:----------------------:|
| Name        |     &check;      |     &check;     | &check; |        &cross;         |
| Tag         |     &check;      |     &check;     | &cross; |        &cross;         |
| Realm       |     &check;      |     &check;     | &cross; |   &check;  (`realm`)   |
| Attribute   |     &check;      |     &check;     | &cross; |        &check;         |
| RealmPrefix |     &check;      |     &cross;     | &cross; |        &cross;         |

It is possible to create own objects using the interfaces:
- `Attacher`: attach information to a logger
- `Condition`: to be usable as condition in a rule.

Only objects implementing at least one of those interfaces can
usefully be passed.

## Bound and Unbound Loggers

By default, logging contexts provide *bound* loggers. The activation of
such a logger is bound to the settings of the rule matching at the time
of its creation. If it does not match any rule, always context's default
level is used.

This behaviour is fine, als long such a logger is used temporarily, for example
it is created at the beginning of a dedicated call hierarchy, and passed down
the call tree. But it does not show the expected behaviour when stored in and
reused from a long-living variable. If the rule settings are changed
during its lifetime, the activation state is NOT adapted.

Nevertheless, it might be useful store and reuse a configured logger.
Configured means, that is instantiated for a dedicated long living message
context, or with a dedicated name. Such a behaviour can be achieved
by not using a logger but a logging context. Because the context does
not provide logging methods a temporary logger has to be created
on-the-fly for issuing log entries.

Another possibility is to use *unbound* loggers created with a message context
for a logging context using the `DynamicLogger` function. It provides
a logger, which keeps track of the actual settings of the context it has been
created for. Whenever the configuration changes, the next logging call will
adapt the effectively used logger on-the-fly. Such loggers keep track of the
context settings as well as the configured message context and logger values
or names (provided by the methods `WithValues` and `WithName`).

They can be used, for example for permanent worker Go routines, to
statically define the log name or standard values used for all subsequent log
requests according to the identity of the worker.

## Support for special logging systems

The general *logr* logging framework acts as a wrapper for
any other logging framework to provide a uniform frontend,
which can be based on any supported base.

To support this, an adapter must be provided, for example,
the adapter for *github.com/sirupsen.logrus* is provided
by *github.com/bombsimon/logrusr*.

Because this logging framework is based on *logr* 
it can be based on any such supported logging framework.

This library contains some additional special mappings of *logr*, also.

### `logrus`

The support includes three new logrus 
entry formatters in package `logrusfmt`, able to be configurable to best match the features of this library.

- `TextFormatter` an extended logrus.TextFormatter with
  extended capabilities to render an entry.
  This is used by the adapter to generate more human-readable
  logging output supporting the special fields provided by
  this logging system.

- `TextFmtFormatter` an extended `TextFormatter` able
  to render more human-readable log messages by 
  composing a log entry's log message incorporating selected 
  log fields into a readable log message.

- `JSONFormatter` an extended logrus.JSONFormatter with
  extended capabilities to render an entry.
  This is used by the adapter to generate more readable
  logging output with a dedicated ordering of the special fields 
  provided by this logging system.

The package `logrusl` provides configuration methods to 
achieve a `logging.LogContext` based on *logrus* with special 
preconfigured configurations.