
# Repository `OCIRegistry` and `ociRegistry` - OCI Registry 


### Synopsis

```
type: OCIRegistry/v1
```

### Description

Artifact namespaces/repositories of the API layer will be mapped to an OCI
registry according to the [OCI distribution specification](https://github.com/opencontainers/distribution-spec/blob/main/spec.md).

Supported specification version is `v1`.

### Specification Versions

#### Version `v1`

The type specific specification fields are:

- **`baseUrl`** *string*

  OCI repository reference

- **`legacyTypes`** (optional) *bool*

  OCI repository requires Docker legacy mime types for OCI
  image manifests. (automatically enabled for docker.io)](OCM component versions can be stored in OCI registries which
are conforming to the [OCI distribution specification](https://github.com/opencontainers/distribution-spec/blob/main/spec.md).
Additionally, a registry must support a deep repository structure.
)

### Go Bindings

The Go binding can be found [here](type.go)
