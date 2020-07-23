# Component Definitions

A Component Definition describes the steps that are necessary to deploy a component. 
These steps are described by either Executors or references to other Component Definitions.

## Executors and Aggregations

### Executors

A Definition may contain any number of Executors. They are provided as one single string in `.executors`. While processing the Definition, the string is templated, after which it should be a valid YAML list, and then stored in the cluster as an `Execution` CR. Executors are processed in the given order, but in parallel with referenced Definitions. See the [Executor documentation](Executors.md) for further information.

```yaml
executors: |
- name: deploy-chart
  type: helm
  config:
    chartRepository: my-repo
    version: 1.0.0
    values: {{ .exports.mykey.x }}
    valueRef:
      secretRef: abc
```

*Example*


### Aggregations

A Definition can aggregate any number of other Definitions by referencing them in `.definitionsRefs`.
To map the imports of the surrounding Definition to their inner definitions, a mapping is needed.
The mapping can be defined for each component for their imports and exports.

```yaml
definitionRefs:
- ref: my-sub-component:1.0.0
  imports:
  - from: abc
    to: yxz
  exports:
  - from: abc
    to: yxz
```

*Example*


## Installations

Component Definitions are not deployed into the cluster. Instead, a Component Installation is deployed which references the corresponding Definition. 
If the referenced Definition aggregates other definitions, their corresponding Installations will be created automatically, the user only needs to deploy the top-level Installation(s). 
See the [documentation on Installations](Installations.md) for details.

## Registry

Component Defintions are referenced by installations with their unique name and version.
This name in combination with the version is then used by the landscaper to fetch the Definition from a registry.

Currently, a local and OCI registry are supported.
Be aware that a local reigstry should be only used for testing and development, whereas the OCI registry is the preferred productive way.

### Local

A local registry can be defined in the landscaper configuration by providing the below configuration.
The landscaper expects the given paths to be a directory that contains the definitions in subdirectory.
The subdirectory should contain the file `description.yaml`, that contains the actual ComponentDefinition with its version and name.
The whole subdirectory is used as the blob content of the Component.
```yaml
apiVersion: config.landscaper.gardener.cloud/v1alpha1
kind: LandscaperConfiguration

registry:
  local:
    paths:
    - "/path/to/definitions"
```

### OCI

ComponentDefinitions can be stored in a OCI compliant registry which is the preferred way to create and offer CompoentnDefinitions.
The Landscaper uses [OCI Artifacts](https://github.com/opencontainers/artifacts) which means that a OCI compliant registry has to be used.
For more information about the [OCI distribution spec](https://github.com/opencontainers/distribution-spec/blob/master/spec.md) and OCI compliant registries refer to the official documents.

The OCI manifest are stored in the below format in the registry.
Whereas the config contains the actual `ComponentDefinition` and the layers 
 - must contain a componentdescriptor of type `application/vnd.gardener.componentdescriptor.v2+json`
 - can contain a content blob of type `application/tar+gzip`
 
 The layers can be identified via their title annotation or via their media type as only one component descriptor per layer is allowed.
```json
{
   "schemaVersion": 2,
   "annotations": {},
   "config": {
      "digest": "sha256:6f4e69a5ff18d92e7315e3ee31c62165ebf25bfa05cad05c0d09d8f412dae401",
      "mediaType": "application/vnd.gardener.landscaper.componentdefinition.v1+json",
      "size": 452
   },
   "layers": [
      {
         "digest": "sha256:6f4e69a5ff18d92e7315e3ee31c62165ebf25bfa05cad05c0d09d8f412dae401",
         "mediaType": "application/vnd.gardener.componentdescriptor.v2+json",
         "size": 420,
         "annotations": {
            "org.opencontainers.image.title": "componentdescriptor"
         }
      },
      {
         "digest": "sha256:6f4e69a5ff18d92e7315e3ee31c62165ebf25bfa05cad05c0d09d8f412dae401",
         "mediaType": "application/tar+gzip",
         "size": 78343,
         "annotations": {
            "org.opencontainers.image.title": "content"
         }
      }
   ]
}
```
