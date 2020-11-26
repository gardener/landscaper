# Container Deployer

The container deployer is a controller that reconciles DeployItems of type `landscaper.gardener.cloud/container`.
It executes the given container spec with the injected imports and collect exports from a predefined path.

### Configuration
This sections describes the provider specific configuration
```yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: DeployItem
metadata:
  name: my-custom-container
spec:
  type: landscaper.gardener.cloud/container

  target: # has to be of type landscaper.gardener.cloud/kubernetes-cluster
    name: my-cluster
    namespace: test

  config:
    apiVersion: container.deployer.landscaper.gardener.cloud/v1alpha1
    kind: ProviderConfiguration

    blueprintRef: <abc....>
    
    importValues: 
      {{ toJson . | indent 2 }}

    image: <image ref>
    command: ["my command"]
    args:  ["--flag1", "my arg"]
```

### Contract

In order for the container deployer to interact with the landscaper a contract for imports, exports and the state has to be defined.

- The current operation that the image should execute is defined by the env var `OPERATION` which can be `RECONCILE` or `DELETE`.
- *Imports* can be expected as a json file at the path given by the env var `IMPORTS_PATH`.
- *Exports* should be written to a json or yaml file at the path given by the env var `EXPORTS_PATH`.
- The optional *state* should be written to the directory given by the env var `STATE_PATH`.
- The complete state directory will be tarred and managed by the landscaper(:warning: no symlinks)
- The *Component Descriptor* can be expected as a json file at the path given by the env var `COMPONENT_DESCRIPTOR_PATH`.
  The json file contains a resolved component descriptor list which means that all transitive component descriptors are included in a list.
  ```json
  {
    "meta":{
      "schemaVersion": "v2"
    },
    "components": [
      {
        "meta":{
          "schemaVersion": "v2"
        },
        "component": {}
      }
      ...
    ]
  }
  ```
- The optional *content blob* that can be defined by a definition can be accessed at the directory given by the env var `CONTENT_PATH`.

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