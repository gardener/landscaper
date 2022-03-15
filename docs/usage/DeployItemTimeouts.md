# DeployItem Timeouts

If not deactivated in the configuration, the landscaper checks for three different timeouts on deployitems:

## Types of Timeouts

#### Pickup Timeout

A pickup timeout occurs if no deployer reacts to changes to a deployitem. Whenever the landscaper creates or updates a deployitem from an execution, it adds a reconcile timestamp annotation (`landscaper.gardener.cloud/reconcile-time`) with the current time. It is part of the [deployer contract](../technical/deployer_contract.md) that the deployer which is responsible for the deployitem has to remove the annotation. If that doesn't happen within a given timeframe, a pickup timeout occurs.

##### Effects

A pickup timeout causes the Landscaper to put the respective deployitem in phase `Failed`. The `.status.lastError` field will contain the reason `PickupTimeout`.

##### Configuration and Default

The timespan can be configured in the Landscaper config by setting `deployItemTimeouts.pickup`.

The default is 5 minutes.


#### Progressing Timeout

A progressing timeout occurs if the deployitem takes too long in phase `Progressing`. To determine the duration for which the deployitem has been progressing, the `.status.lastReconcileTime` field is used. Deployers are required to set this field to the current time when they start progressing a deployitem, see the [deployer contract](../technical/deployer_contract.md).

##### Effects

A progressing timeout causes the Landscaper to abort the deployitem by adding the [abort operation annotation](../usage/Annotations.md) to it. It will also add the abort timestamp annotation, which is required for the aborting timeout.

##### Configuration and Default

There are two possibilities to configure the progressing timeout for a deployitem:
- The timespan can be configured per deployitem using the deployitem's `.spec.timeout` field.
- If not configured in the deployitem, the default from the Landscaper config is used. It can be set via the `deployItemTimeouts.progressingDefault` field.

If not overwritten, the `.spec.timeout` field is empty and the default in the Landscaper config is set to 10 minutes.


#### Aborting Timeout

An aborting timeout occurs if the deployer takes too long to abort a deployitem. This is checked via an abort timestamp annotation (`landscaper.gardener.cloud/abort-time`) which is set by the Landscaper when a progressing timeout occurs. Note that this means that an aborting timeout can currently only occur after a progressing timeout has occurred, not after a deployitem has been aborted manually (unless the annotation has also been set manually).

##### Effects

An aborting timeout causes the Landscaper to put the respective deployitem in phase `Failed`. The `.status.lastError` field will contain the reason `AbortingTimeout`.

##### Configuration and Default

The timespan can be configured in the Landscaper config by setting `deployItemTimeouts.abort`.

The default is 5 minutes.


## Configuring the Timeouts

The timeouts can be configured in the Landscaper config. The accepted values are `none` and everything that is parsable by golang's `time.ParseDuration()` function.

Note that for the progressing timeout, the `.spec.timeout` field in the deployitem takes precedence over the corresponding value in the Landscaper config.

To deactivate timeout checking altogether, set the timespan for the respective timeout to `none` (or to a duration that is equivalent to zero seconds).

**Example**
```yaml
landscaper:
  deployers: ...
  deployerManagement: ...

  deployItemTimeouts:
    pickup: 30s
    abort: none
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
      abort: none
      progressingDefault: 1h
```