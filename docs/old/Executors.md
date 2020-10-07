# Executors

There are 3 different execution types:
- Container
- Script
- Templates

These execution types are internally translated into a `DeployItem` CR.
The Deployers communicate with the landscaper through this dedicated DeployItem CR.
These DeployItems are the extensions of the landscaper which means that they execute the actual components.

By default the landscaper is deployed with 2 default Deployers: Container and Script.
Other Deployers can be used with the templates execution type that templates such DeployItems.

Imports and Exports are synced between the deployer and the landscaper via secrets.

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: my-deploy-item
spec:

  type: container | helm | manifest | ...

  import:
    secretRef: 
      name: secret-item1
      key: abc

  providerConfig:
    xxx: abc

status:
  phase: Reconciled
  conditions:
  - condition1: healthy
    message: testing

  exportGeneration: 2

  exports:
    value: xxx
    secretRef: 
      name: secret-item3-exp
      key: abc
```

## Container and Script Deployer

The container/script deployer is a Deployer that handles DeployItems of type `container` and `script`.

If the landscaper creates or updates a container/script DeployItem the deployer creates a dedicated pod 
that executes the described config of the item.
The import configuration is synced to the pods filesystem via a secret (created by the landscaper and referenced by the DeployItem CR).

The output configuration is collected by a sidecar that watches a specific path (env var `EXPORT_PATH`).
As soon as the actual workload has finished the sidecar reads the config from the export path and creates or updates the referenced export secret.

State: the state of a container or script is persisted by copying data to a specific directory (envvar `STATE`).
This directory is then tarred and persisted as secret data.
In the future we may have to use volumes or a blob store.


## Constraints

### State Handling

Each Deployer is responsible for persisting and managing the state of DeployItems of its type.

### Idempotence

DeployItems might be processed multiple times for various reasons, their execution is therefore required to be idempotent.