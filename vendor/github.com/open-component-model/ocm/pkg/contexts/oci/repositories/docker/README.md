
# Repository `DockerDaemon` - Images stored in a Docker Daemon


### Synopsis

```
type: DockerDaemon/v1
```

### Description

This repository type provides a mapping of the image repository behind a docker
daemon to the OCI registry access API.

This is only possible with a set of limitation:
- It is only possible to store and access flat images
- There is no access by digests, only by tags.
- The docker image id can be used as pseudo digest (without algorithm)

### Specification Versions

#### Version `v1`

The type specific specification fields are:

- **`dockerHost`** *string*

  Address of the docker daemon to use.

### Go Bindings

The Go binding can be found [here](type.go)
