
# Repository `OCIRegistry` and `ociRegistry` - OCI Registry based OCM Repositories


### Synopsis

```
type: OCIRegistry/v1
```

### Description

The content of the OCM repository will be stored in an OCI registry using
a dedicated OCI repository name prefix.

Supported specification version is `v1`.



### Specification Versions

#### Version `v1`

The type specific specification fields are:

- **`baseUrl`** *string*

  OCI repository reference, containing the host part and the repository prefix
  as path

- **`legacyTypes`** (optional) *bool*

  OCI repository requires docker legacy mime types for OCI
  image manifests. (automatically enabled for docker.io)

  `docker.io` cannot be used to host an OCM repository because 
  is provides a fixed number of levels for repository names (2).


### Go Bindings

The Go binding can be found [here](../../../oci/repositories/ocireg/type.go).
