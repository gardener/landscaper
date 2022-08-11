# Logging Guidelines

## Design

### Goals

1. Logging should be consistent among all Landscaper components. Logs should always look the same.
2. Logging should be intuitive for users: It should be clear what a log message is saying and whether it is an error or not.
3. Logging should be intuitive for developers: It should be clear which log level/verbosity to use for a specific log output.
4. The logging framework should be able to provide structured logging as `json`, so it can be interpreted by third-party logging analytics software.
5. The logr logging framework is used by large parts of the community. Compatibility is desired.

### Guidelines

#### Getting a Logger

In most cases, creating a new logger is not required, as it can be fetched from the `context.Context` object. This can be done by using `logging.FromContext(ctx)`. If that fails, the best way is to call the singleton-style function `logging.GetLogger()`. It will return an existing logger, if the function has been used before, and create a new logger otherwise. In the latter case, the new logger will be initialized with default configuration, which takes into account any logging-specific flags given to the executable - this is usually the desired configuration.

Apart from that, the `logging.New(config)` and `logging.NewCLILogger()` functions provide further options for constructing new loggers. Both are specific to the underlying logging implementation and will probably change if it is swapped. The former one requires a logging configuration which will be defaulted if set to `nil`, while the latter one uses a default configuration which is suitable for a CLI.

To store an existing logger in a context, construct a new context using `logging.NewContext(ctx, logger)`.

#### Logging Resource Reconciliations

Most often, k8s controllers watch a specific resource kind and react on changes to objects of this kind. The logging framework has some methods to support unified log messages among multiple controllers:

When setting up a logger for a controller, it is adviced to use the `log.Reconciles(name, kind)` method. This is basically a wrapper for `log.WithName(name).WithValues("reconciledResourceKind", kind)`. Use the singular form for both name and kind, with name being lower camel case and kind being upper camel case.

At the beginning of a controller's `Reconcile` function, call either `logging.StartReconcileFromContext(ctx, req)` (if 
you don't have a logger available, this will try to fetch one from the context) or `log.StartReconcile(req)` (if you 
already have a logger), with `req` being the `reconcile.Request` object. The returned logger will then contain a 
key-value-pair with the namespaced name of the reconciled object. In addition, the function will print a log message 
that a new reconciliation has begun. Use `log.StartReconcileAndAddToContext(ctx, req,...)` if you need a new context 
containing the new logger. `logging.MustStartReconcileFromContext` combines this method with `FromContextOrNew`, making the task of getting the logger from the context (or creating a new one, if that fails), adding a key-value-pair for the reconciled resource as well as further configurable pairs, and logging the beginning of a new reconciliation a one-liner.

#### Conventions for Names, Keys, and Messages

Use lower camel case for names given to `log.WithName(name)` and for keys of key-value-pairs. Duplicate keys should be avoided.

In `github.com/gardener/landscaper/controller-utils/pkg/logging/constants`, there are constants for commonly used keys and log messages. Before using a custom string, please check whether a fitting constant exists and use it instead, if that's the case. If not, consider adding one, if the key/message is expected to be used in multiple locations in the code.

Log messages should start with a capital letter and not end with a punctuation mark.

#### Verbosity Levels and what to log

The logging framework supports three verbosity levels - _error_, _info_, and _debug_ (see below for the reasoning behind this decision). These are some hints for when to use which verbosity for printing a log message.

##### Error

The `error` verbosity should only be used for errors. Since most failures cause actual errors (rather than just logs), which don't need to be logged in this way, logs at `error` verbosity will mostly be used for unexpected failures which can be recovered from, and they should be relatively rare.

##### Info

Logs logged at `info` verbosity should try to provide sufficient information for a reader to find out what the component is doing, but at the same time, the logs should remain readable and not flooded with messages. The amount of `info` logs follows the principle 'as few as possible, but as much as necessary'. Since this is the verbosity at which the Landscaper prints by default, known problems (e.g. wrong configuration) should be identifyable easily with only these logs. 

##### Debug

`debug` is the most verbose log level. As such, everything which is not an error and too verbose for `info` should be logged as a `debug` message. A user who reads these logs wants to have precise knowledge about what happened and which code paths were used, with the price being having to read logs of log messages. The results of important conditional statements should be visible, preferrably with the value which caused this decision. To not flood the already verbose `debug` logs, try to avoid log statements which don't add new meaningful information.

## The Framework

### logging.Logger vs. logr.Logger

The `logr` logging framework is widespread among the kubernetes community. However, it cannot be easily expanded with new methods, which is why we decided to build our own logging framework. The main struct `Logger` is implemented in `controller-utils/pkg/logging/logger.go` and it is basically a _stateless wrapper around a logr.Logger_. It uses an `logr.Logger` internally, so the actual logging implementation can be swapped out easily. Since it doesn't store anything else except for the internal logger, one can convert between `logr.Logger` and `logging.Logger` easily and at any time without losing information.
```go
var log logging.Logger
communityLogger := log.Logr() // .Logr() gives the internal logr.Logger
log = logging.Wrap(communityLogger) // logging.Wrap(l) wraps a logging.Logger around the given logr.Logger l
```

### Logging with the Logger

Furthermore, `logging.Logger` has nearly all methods which are available for `logr.Logger` and they work exactly the same, in most cases. The most notable exceptions are caused by our decision to not use integers for verbosity.

We decided against integer values for verbosity, because it is hard to define which value is to be used for which log output. In the end, this will most likely lead to inconsistent logging verbosity among the Landscaper as each developer uses his/her gut feeling when writing logs. But also the user would never be sure which verbosity needs to be enabled to see the desired logs.
For this reason, the logging framework uses only three distinct verbosity levels: `error`, `info`, and `debug` (from least to most verbose). Code-wise, the three levels are represented as enum-like constants:

```go
var ll logging.LogLevel
ll = logging.DEBUG
ll = logging.INFO
ll = logging.ERROR
```

To log at a specific level, simply use the method of the same name. As for `logr.Logger`, all of these methods optionally take an arbitrary number of key-value-pairs.
```go
log.Debug("this is a debug message")
log.Info("this is an info message") // same as logr.Info
log.Error(err, "this is an error message") // same as logr.Error
```

The `Log` method can be used to log at a dynamically defined verbosity level. Note that it is not possible to pass an `error` object for logs at `ERROR` level in this case.
```go
myLogLevel := logging.DEBUG
log.Log(myLogLevel, "this is a message with a dynamic log level")
```

Due to the aforementioned logging methods, there is no need for a `log.V(myVerbosity)` method as it exists within the `logr` framework. As a result, our `log.Enabled()` method takes a log level as an argument:
```go
log.Enabled(logging.DEBUG) // will return true if log is configured to print logs at debug level
```
