# Configuring the Landscaper Logs

There are multiple flags which can be used to control the way in which the Landscaper prints its logs.

Two of them influence multiple settings at once:
  - **--cli**: This flag is useful, if the logs are printed directly to a console. It enables colors for the verbosity levels and removes the timestamp. Also, the default encoding is `text`.
  - **--dev**: Activates some features which are useful when developing on the Landscaper, which includes enabling callers and stacktraces, as well as defaulting the verbosity to `debug`.

The other flags which influence the logging behavior are more fine-grained and modify only one characteristic each. If they are set, they will override potential defaults set by one of the flags above:
  - **--disable-caller**: Whether printing the filename and line number of the code statement causing the log should disabled. Defaults to `true`. Influenced by the `--dev` flag.
  - **--disable-stacktrace**: Whether printing the stacktrace for logs on `error` level should be disabled. Defaults to `true`. Influenced by the `--dev` flag.
  - **--disable-timestamp**: Whether printing the timestamps should be disabled. Defaults to `false`. Influenced by the `--cli` flag.
  - **--format**: How the logging output should be formatted. Valid values are `text` and `json`. Default is `json`, but both, the `--cli` and the `--dev` flag will default this to `text`.
  - **--verbosity**: At which verbosity logs should be printed. Valid values are `error`, `info`, and `debug` (sorted from least to most verbose). Default is `info`, unless the `--dev` flag ist used, then it is `debug`.

The defaults are chosen in a way that results in a production-ready logging configuration, if none of the logging flags is set. The result will be JSON logging at 'info' level, with timestamps and without callers and stacktraces.