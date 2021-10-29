# Mock Deployer

The mock deployer is a controller that reconciles DeployItems of type `landscaper.gardener.cloud/mock`. 

This deployer is only ment for testing and demo purposes to simluate specific behavior of deploy item. Therefore, the Configuration part configures the state that should be reconciled by the mock.

**Index**:
- [Provider Configuration](#provider-configuration)
- [Provider Status](#status)
- [Deployer Configuration](#deployer-configuration)

### Provider Configuration

This sections describes the provider specific configuration

```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: my-nginx
spec:
  type: landscaper.gardener.cloud/mock

  config:
    apiVersion: mock.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration
    # Specifies the phase of this DeployItem
    phase: Init
    # Specifies the provider specific status
    providerStatus:
      apiVersion: mock.deployer.landscaper.gardener.cloud/v1alpha1
      kind: ProviderStatus
      key1: val1
    # Specifies the exported data that will be reconciled into the exportRef.
    export:
      key2: val2
#    continuousReconcile: # configure continuous reconciliation
#      every: "1h"
#      cron: cron: "* */1 * * *"

```

##### Continuous Reconciliation
For information on the continuous reconciliation configuration, see [here](../development/deployer-extensions##continuous-reconcile-extension) under 'usage'.

### Status

The status is reconciled as defined in the configuration.

## Deployer Configuration

When deploying the mock deployer controller it can be configured using the `--config` flag and providing a configuration file.

The structure of the provided configuration file is defined as follows.

:warning: Keep in mind that when deploying with the helm chart the configuration is abstracted using the helm values. See the [helm values file](../../charts/mock-deployer/values.yaml) for details when deploying with the helm chart.
```yaml
apiVersion: mock.deployer.landscaper.gardener.cloud/v1alpha1
kind: Configuration

# target selector to only react on specific deploy items.
# see the common config in "./README.md" for detailed documentation.
targetSelector:
  annotations: []
  labels: []
```
