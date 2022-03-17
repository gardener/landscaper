# Remote Access

Blueprints are referenced in installations or installation templates via the component descriptors access.

Basically blueprints are a filesystem, therefore, any blob store could be supported.
Currently, local and OCI registry access is supported.

:warning: Be aware that a local registry should be only used for testing and development, whereas the OCI registry is the preferred productive method.


## Local

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

The blueprints are referenced via `local` access type in the component descriptor.
```
component:
  localResource:
  - name: blueprint
    type: blueprint
    access:
      type: local
```

## OCI

ComponentDefinitions can be stored in a OCI compliant registry which is the preferred way to create and offer ComponentDefinitions.
The Landscaper uses [OCI Artifacts](https://github.com/opencontainers/artifacts) which means that a OCI compliant registry has to be used.
For more information about the [OCI distribution spec](https://github.com/opencontainers/distribution-spec/blob/master/spec.md) and OCI compliant registries refer to the official documents.

The OCI manifest is stored in the below format in the registry.
Whereas the config is ignored and there must be exactly one layer with the containing a bluprints filesystem as `application/tar+gzip`.
 
 The layers can be identified via their title annotation or via their media type as only one component descriptor per layer is allowed.
```json
{
   "schemaVersion": 2,
   "annotations": {},
   "config": {},
   "layers": [
      {
         "digest": "sha256:6f4e69a5ff18d92e7315e3ee31c62165ebf25bfa05cad05c0d09d8f412dae401",
         "mediaType": "application/tar+gzip",
         "size": 78343,
         "annotations": {
            "org.opencontainers.image.title": "definition"
         }
      }
   ]
}
```

The blueprints are referenced via `ociRegistry` access type in the component descriptor.
```
component:
  localResource:
  - name: blueprint
    type: blueprint
    access:
      type: ociRegistry
      imgageReference: oci-ref:1.0.0
```
