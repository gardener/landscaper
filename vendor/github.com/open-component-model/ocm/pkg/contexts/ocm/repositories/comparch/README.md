
# Repository `ComponentArchive` - Filesystem based Storage of a Component Version


### Synopsis

```
type: ComponentArchive/v1
```

### Description

The content of a single OCM Component Version will be stored as Filesystem content.
This is a special version of an OCM Repository, which can be used to
compose a component version during the build time of a component.

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
  - `tgz`: stored as directory structure in a tar file compressed by GNU Zip


### Go Bindings

The Go binding can be found [here](type.go).
