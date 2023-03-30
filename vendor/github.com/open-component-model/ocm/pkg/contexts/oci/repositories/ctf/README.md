
# Repository `CommonTransportFormat` - Filesystem-based Storage of OCI repositories


### Synopsis

```
type: CommonTransportFormat/v1
```

### Description

Artifact namespaces/repositories of the API layer will be mapped to a
filesystem-based representation according to the [Common Transport Format specification](formatspec.md).

Supported specification version is `v1`.

### Specification Versions

#### Version `v1`

The type specific specification fields are:

- **`filePath`** *string*

  The path in the filesystem used to store the content

- **`fileFormat`** *string*

  The file format to use:
  - `directory`: stored as file hierarchy in a directory
  - `tar`: stored as file hierarchy in a TAR file
  - `tgz`: stored as file hierarchy in a GNU-zipped TAR file (tgz)
  
- **`accessMode`** (optional) *byte*

  Access mode used to access the content:
  - 0: write access
  - 1: read-only
  - 2: create id not existent, yet
  
### Go Bindings

The Go binding can be found [here](type.go)