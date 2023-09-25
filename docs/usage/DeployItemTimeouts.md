# DeployItem Timeouts

If not deactivated in the configuration, the landscaper checks for two different timeouts on deploy items: pickup and
progressing timeout.

## Pickup Timeout

The Landscaper marks a deploy item as processable by a deployer by setting its status field `jobID` to a new value 
which differs from the status field `jobIDFinished`. During this update is also sets the status field `jobIDGenerationTime`
on the current timestamp.

When a deployer picks up a deploy item it sets the status field `lastReconcileTime` on its current time. This is part 
of the [deployer contract](../technical/deployer_contract.md).

A pickup timeout occurs if no deployer reacts to a deploy item which should be processed within the configured timeframe.

### Effects

A pickup timeout causes the Landscaper to set the status fields `Phase` and `DeployItemPhase` of the deploy item on 
on `Failed` and sets the `finishedJobID` on the value of `jobID`. The `.status.lastError` field 
will contain the reason `PickupTimeout`.

### Configuration and Default

The timespan can be configured in the Landscaper config by setting `landscaper.deployItemTimeouts.pickup`.

The default is 5 minutes.

The accepted values are `none` and everything that is parsable by golang's `time.ParseDuration()` function.
To deactivate this timeout check altogether, set the timespan for the respective timeout to `none`
(or to a duration that is equivalent to zero seconds).

**Example**
```yaml
landscaper:
  deployers: ...
  deployerManagement: ...

  deployItemTimeouts:
    pickup: 30s
```

Please note that the above configuration has to be wrapped in another `landscaper` node, if it is set via the Landscaper's helm chart's values:
```yaml
landscaper:
  image: ...

  landscaper:
    deployers: ...
    deployerManagement: ...

    deployItemTimeouts:
      pickup: 30s
```


## Progressing Timeout

A progressing timeout occurs if the deploy item has been picked by a deployer, but it does not finish its job within some
timeframe.

### Effects

A progressing timeout causes the Landscaper to put the status fields `phase` of the deploy item
on `Failed` and sets the `finishedJobID` on the value of `jobID`. The `lastError` field will contain the reason
`ProgressingTimeout`.

### Configuration and Default

There are two possibilities to configure the progressing timeout for a deploy item:
- The timespan can be configured per deploy item using the deploy item's `.spec.timeout` field.
- If not configured in the deploy item, the timeout defaults to 10 minutes.
