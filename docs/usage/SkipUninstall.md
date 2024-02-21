---
title: Skip Uninstallation
sidebar_position: 12
---

# Skipping the Uninstallation of an Application

When you delete an Installation or remove a DeployItem from an Installation, normally, Landscaper will uninstall the 
corresponding application from the target cluster. However, if the target cluster is unreachable, the uninstallation is 
impossible, so that the DeployItem and the Installation will get into the phase `Failed`, resp. `DeleteFailed`. 
To avoid this, you can configure for a DeployItem to skip the uninstallation if the target cluster does not exist 
or has a deletion timestamp.

Skipping the uninstallation in case of a missing cluster is not the standard behavior, because the existence check 
requires some prerequisites. In general, the Landscaper cannot distinguish whether a cluster has been deleted or 
is unavailable due to an error.


### Prerequisites

The present feature is only supported if the following prerequisites are all satisfied:

- a [TargetSync][1] object must exist in the namespace of the Installation,
- the Target of the DeployItem must have been created by this TargetSync.


### Procedure

DeployItems are defined by a template in a blueprint.
To enable the present feature for a DeployItem, add the field `.onDelete.skipUninstallIfClusterRemoved` to the 
DeployItem template and set its value to `true`. Here is an example:

```yaml
deployExecutions:
  - name: default
    type: GoTemplate
    template: |
      deployItems:
        - name: item1
          type: landscaper.gardener.cloud/kubernetes-manifest
          target:
            import: cluster1
          onDelete:
            skipUninstallIfClusterRemoved: true
          config:
            ...
```

If a blueprint has several DeployItems, the `skipUninstallIfClusterRemoved` setting can be specified for each
of them individually.


### Resulting Behavior

Suppose a DeployItem has been configured as described above. When the Installation or DeployItem is being deleted,
Landscaper will check the Gardener Shoot resource of the target cluster. If it does not exist or has a deletion timestamp,
Landscaper will skip the uninstallation of the deployed application, and will directly delete the DeployItem. 


<!-- References -->

[1]: TargetSyncs.md  
