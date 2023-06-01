
# Repository `CommonTransportFormat` - Filesystem based OCM Repositories


### Synopsis

```
type: CommonTransportFormat/v1
```

### Description

The content of an OCM repository will be stored on a filesystem.

Supported specification version is `v1`.

### Specification Versions

#### Version `v1`

The type specific specification fields are:

- **`filePath`** *string*

  Path in filesystem used to host the repository.

- **`fileFormat`** (optional) *string*

  The format to use to store content:
  - `directory`: stored as directory structure
  - `tar`: stored as directory structure in a tar file
  - `tgz`: stored as directory structure in a tar file compressed by GNU Zip.


### Go Bindings

The Go binding can be found [here](../../../oci/repositories/ctf/type.go).
