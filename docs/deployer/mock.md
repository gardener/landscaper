# Mock Deployer

The mock deployer is a controller that reconciles DeployItems of type `Mock`.<br>
This deployer is only ment for testing and demo purposes to simluate specific behavior of deploy item.
Therefore, the Configuration part configures the state that should be reconciled by the mock.

### Configuration
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
```
### Status
The status is reconciled as defined in the configuration