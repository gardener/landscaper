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
  type: Mock
  importRef:
    name: secret-item1
    namespace: default

  config:
    # Specifies the phase of this DeployItem
    phase: Init
    # Specifies the provider specific status
    providerStatus:
      key1: val1
    # Specifies the exported data that will be reconciled into the exportRef.
    export:
      key2: val2
```
### Status
The status is reconciled as defined in the configuration