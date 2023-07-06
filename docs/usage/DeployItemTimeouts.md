# DeployItem Timeouts

If not deactivated in the configuration, the landscaper checks for two different timeouts on deployitems:

## Types of Timeouts

#### Pickup Timeout

The Landscaper marks a deploy item as processable by a deployer by setting its status field `jobID` to a new value 
which differs from the status field `jobIDFinished`. During this update is also sets the status field `jobIDGenerationTime`
on the current timestamp.

When a deployer picks up a deploy item it sets the status field `lastReconcileTime` on its current time. This is part 
of the[deployer contract](../technical/deployer_contract.md).

A pickup timeout occurs if no deployer reacts to a deploy item which should be processed within the configured timeframe.

##### Effects

A pickup timeout causes the Landscaper to set the status fields `Phase` and `DeployItemPhase` of the deploy item on 
on `Failed` and sets the `finishedJobID` on the value of `jobID`. The `.status.lastError` field 
will contain the reason `PickupTimeout`.

##### Configuration and Default

The timespan can be configured in the Landscaper config by setting `deployItemTimeouts.pickup`.

The default is 5 minutes.

#### Progressing Timeout

A progressing timeout occurs if the deploy item has been picked by a deployer but it does not finish its job within some
timeframe.

##### Effects

A progressing timeout causes the Landscaper to put the status fields `phase` of the deployitem
on `Failed` and sets the `finishedJobID` on the value of `jobID`. The `lastError` field will contain the reason
`ProgressingTimeout`.

##### Configuration and Default

There are two possibilities to configure the progressing timeout for a deployitem:
- The timespan can be configured per deployitem using the deployitem's `.spec.timeout` field.
- If not configured in the deployitem, the default from the Landscaper config is used. It can be set via 
  the `deployItemTimeouts.progressingDefault` field, resp. defaults to 10 minutes.


## Configuring the Timeouts

The timeouts can be configured in the Landscaper config. The accepted values are `none` and everything that is parsable 
by golang's `time.ParseDuration()` function.

Note that for the progressing timeout, the `.spec.timeout` field in the deployitem takes precedence over the corresponding 
value in the Landscaper config.

To deactivate timeout checking altogether, set the timespan for the respective timeout to `none` (or to a duration that 
is equivalent to zero seconds).

**Example**
```yaml
landscaper:
  deployers: ...
  deployerManagement: ...

  deployItemTimeouts:
    pickup: 30s
    progressingDefault: 1h
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
      progressingDefault: 1h
```