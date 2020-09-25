# Container Deployer

The container deployer is a controller that reconciles DeployItems of type `Container`.
It executes the given container spec with the injected imports and collect exports from a predefined path.

### Configuration
This sections describes the provider specific configuration
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: my-custom-container
spec:
  type: Container

  config:
    apiVersion: container.deployer.landscaper.gardener.cloud
    kind: ProviderConfiguration

    blueprintRef: <abc....>

    image: <image ref>
    command: ["my command"]
    args:  ["--flag1", "my arg"]
```

### Contract

In order for the container deployer to interact with the landscaper a contract for imports, exports and the state has to be defined.

The current operation that the image should execute is defined by the env var `OPERATION` which can be `RECONCILE` or `DELETE`.<br>
*Imports* can be expected as a json file at the path given by the env var `IMPORTS_PATH`.<br>
*Exports* should be written to a json or yaml file at the path given by the env var `EXPORTS_PATH`.<br>
The optional *state* should be written to the directory given by the env var `STATE_PATH`.
The complete state directory will be tarred and managed by the landscaper(:warning: no symlinks)<br>
The *Component Descriptor* can be expected as a json file at the path given by the env var `COMPONENT_DESCRIPTOR_PATH`.<br>
The optional *content blob* that can be defined by a definition can be accessed at the directory given by the env var `CONTENT_PATH`.

### Status
This section describes the provider specific status of the resource
```yaml
status:
  providerStatus:
    apiVersion: container.deployer.landscaper.gardener.cloud
    kind: ProviderStatus
    # A human readable message indicating details about why the pod is in this condition.
    message: string
    # A brief CamelCase message indicating details about why the pod is in this state.
    # e.g. 'Evicted'
    reason: string
    # Details about the container's current condition.
    state: corev1.ContainerState
    # The image the container is running.
    image: string
    # ImageID of the container's image.
    imageID: string
```