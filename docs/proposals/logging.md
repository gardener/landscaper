# Logging Design / Framework

This document aims at gathering ideas for improving logging throughout the Landscaper and making it more consistent.

It is partly inspired by the [Gardener Logging Guidelines](https://github.com/gardener/gardener/blob/master/docs/development/logging.md).

## Goals

1. Logging should be consistent among all Landscaper components. Logs should always look the same.
2. Logging should be intuitive for users: It should be clear what a log message is saying and whether it is an error or not.
3. Logging should be intuitive for developers: It should be clear which log level/verbosity to use for a specific log output.
4. The logging framework should be able to provide structured logging as `json`, so it can be interpreted by third-party logging analytics software.
5. The logr logging framework is used by large parts of the community. Compatibility is desired.


## Guideline Proposals

These are some proposals for a logging guideline.

1. Use only three log levels: `error` (logs only errors), `info` (the default), `debug` (logs everything). This way, it's easy for the developer to chose the correct level and someone who wants more logs doesn't have to guess the verbosity levels.

2. Use structured logging (key-value-pairs) instead of `printf`-style logging. This makes it easier to search for specific error messages and process the logs programmatically.

3. Use unified keys for structured logging. This can bey achieved by defining constants and/or specific logging methods implementation-wise. This means that e.g. the name of the resource which is currently being reconciled should always have the key `resource`.

4. Don't construct new loggers unless there is none available. The controller-runtime injects the logger into the `context.Context`, from where it can be fetched during the `Reconcile` call. Since nearly all instances of logging should happen as part of a `Reconcile`, the used logger should always come from some instance of `ctxlogger.WithName(...)`, where `ctxlogger` is the logger derived from the context. It should rarely be necessary to construct a completely new logger.

5. Log messages should start with a capital letter and should not end with a punctuation mark.


## Implementation Idea

Use `logr.Logger` internally, but put a wrapper around it to provide additional/alternative functionality. This allows to abstract commonly used log statements into functions, which will make them easier to use for developers and improve consistency.

Here are some examples, what this could look like:
```go
import (
  "github.com/gardener/landscaper/pkg/utils/logging"
)

func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
  
  log := logging.StartReconcile(ctx, req)
  /*
  Fetches the logger from the context or constructs a new one, if there is none. Then logs the beginning of the reconciliation and returns a logger.
  Equivalent to:
  log := logr.FromContext(ctx)
  if log == nil {
    log = logr.NewLogger()
  }
  log = log.WithValues("resource", req.NamespacedName())
  log.Info("Beginning reconciliation")
  */

  log.Debug("Fetching resource from cluster")
  /*
  log.Debug is a wrapper around the verbosity levels from logr
  Equivalent to:
  log.V(1).Info(...)
  */

  log.WithImport("myimport", lsv1alpha1.ImportTypeData).Error("Import not found")
  /*
  log.WithImport is a wrapper around WithValues
  Equivalent to:
  log.WithValues("import", "myimport").WithValues("importType", lsv1alpha1.DataImport).Error(...)
  */

  log.WithValues(logging.Keys.ReferencedResource, secret.NamespacedName()).Info(...)
  /*
  For cases where we don't have explicit methods, we can provide constants which can be used to increase consistency among log messages.
  Equivalent to:
  log.WithValues("reference", secret.NamespacedName()).Info(...)

  The same concept could also be applied to log messages, if we happen to have the same messages at multiple places in the code, e.g.
  log.Info(logging.Messages.WaitingForSubinstallations)
  */
}
```