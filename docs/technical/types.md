# Types

This documents the resource and media type that are used by the landscaper and somme deployers.

For known target types see [target types](./target_types.md)

Resource type refer to the type defined in a component descriptor.

**Index**:
- [Blueprints](#blueprints)
- [JSON Schema](#json-schema)
- Deployer
  - [Helm Chart](#helm-chart)

### Blueprints
<table>
    <tr>
        <td>ResourceType</td>
        <td> <code>landscaper.gardener.cloud/blueprint</code> (deprecated <code>blueprint</code>) </td>
    </tr>
    <tr>
        <td>Access</td>
        <td> 
            localFilesystemBlob: <code>application/vnd.gardener.landscaper.blueprint.v1+tar+gzip</code> <br>
            Standalone Artifact (ociRegistry): oci artifact with one layer of type <code>application/vnd.gardener.landscaper.blueprint.v1+tar+gzip</code>
        </td>
    </tr>
</table>

**LocalBlob**:

If the blueprint is stored as local blob 

_Example_:
```yaml
resources:
- name: my-blueprint
  type: landscaper.gardener.cloud/blueprint
  access:
    type: localFilesystemBlob
    filename: my-bloc
    mediaType: application/vnd.gardener.landscaper.blueprint.v1+tar+gzip
```

A blueprint blob is expected to be a gzipped tar that MUST contains the `blueprint.yaml` at the root.

**Standalone Artifact**:

The OCI manifest of the Blueprint in this case would look like this:
Whereas the config is ignored and can be anything.
It is recommended to use empty json to comply with most oci registry implementations.

```json
{
 "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
 "schemaVersion": 2, 
 "config": { 
   "digest": "sha256:efg",
   "mediaType": "application/json"
 },
 "layers": [
   {
     "digest": "sha256:efg",
     "mediaType": "application/vnd.gardener.landscaper.blueprint.v1+tar+gzip"
   }
 ]
}
```

```yaml
resources:
- name: my-blueprint
  type: landscaper.gardener.cloud/blueprint
  access:
    type: ociRegistry
    imageReference: <oci artifact uri>
```

### JSON Schema

<table>
    <tr>
        <td>ResourceType</td>
        <td> <code>landscaper.gardener.cloud/jsonschema</code> </td>
    </tr>
    <tr>
        <td>Access</td>
        <td> 
            localFilesystemBlob: <code>application/vnd.gardener.landscaper.jsonscheme.v1+json</code> <br>
            Standalone Artifact (ociRegistry): not yet implemented
        </td>
    </tr>
</table>

**LocalBlob**:

the jsonschema blob is expected to be a stored as plain json of type `application/vnd.gardener.landscaper.jsonscheme.v1+json`

_Example_:
```yaml
resources:
- name: my-schema
  type: landscaper.gardener.cloud/jsonschema
  access:
    type: localFilesystemBlob
    filename: my-bloc
    mediaType: application/vnd.gardener.landscaper.jsonscheme.v1+json
```


## Deployer

### Helm Chart
<table>
    <tr>
        <td>ResourceType</td>
        <td> <code>helm.io/chart</code> (deprecated <code>helm</code>) </td>
    </tr>
    <tr>
        <td>Access</td>
        <td> 
            localFilesystemBlob: <code>application/tar+gzip</code> <br>
            Standalone Artifact (ociRegistry): oci artifact with a config containing the <code>Chart.yaml</code> one layer of type <code>application/tar+gzip</code>
        </td>
    </tr>
</table>

**LocalBlob**:

If the blueprint is stored as local blob

_Example_:
```yaml
resources:
- name: my-helm-chart
  type: helm.io/chart
  access:
    type: localFilesystemBlob
    filename: my-bloc
    mediaType: application/tar+gzip
```

A blueprint blob is expected to be a gzipped tar that MUST contains the `blueprint.yaml` at the root.

**Standalone Artifact**:

The helm deployer supports the default oci helm cartifact as created by the helm cli.
For more information about the usage see the official [helm docs](https://helm.sh/docs/topics/registries/#where-are-my-charts)

```json
{
  "schemaVersion": 2,
  "config": {
    "mediaType": "application/vnd.cncf.helm.config.v1+json",
    "digest": "sha256:8ec7c0f2f6860037c19b54c3cfbab48d9b4b21b485a93d87b64690fdb68c2111",
    "size": 117
  },
  "layers": [
    {
      "mediaType": "application/tar+gzip",
      "digest": "sha256:1b251d38cfe948dfc0a5745b7af5ca574ecb61e52aed10b19039db39af6e1617",
      "size": 2487
    }
  ]
}
```

```yaml
resources:
- name: my-helm-chart
  type: helm.io/chart
  access:
    type: ociRegistry
    imageReference: <oci artifact uri>
```
