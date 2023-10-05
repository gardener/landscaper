
# Access Method `localBlob` - OCM Repository Local Blob Access


### Synopsis
```
type: localBlob/v1
```

Provided blobs use the following media type: attribute `mediaType`

### Description

This method is used to store a resource blob along with the component descriptor
on behalf of the hosting OCM repository.

Its implementation is specific to the implementation of OCM
repository used to read the component descriptor. Every repository
implementation may decide how and where local blobs are stored,
but it MUST provide an implementation for this method.

Regardless of the chosen implementation the attribute specification is
defined globally the same.

Supported specification version is `v1`

### Specification Versions

#### Version `v1`

The type specific specification fields are:

- **`localReference`** *string*

  Repository type specific location information as string. The value
  may encode any deep structure, but typically just an access path is sufficient.

- **`mediaType`** *string*

  The media type of the blob used to store the resource. It may add 
  format information like `+tar` or `+gzip`.

- **`referenceName`** (optional) *string*

  This optional attribute may contain identity information used by
  other repositories to restore some global access with an identity
  related to the original source.

  For example, if an OCI artifact originally referenced using the
  access method [`ociArtifact`](../../../../../docs/formats/accessmethods/ociArtifact.md) is stored during
  some transport step as local artifact, the reference name can be set
  to its original repository name. An import step into an OCI based OCM
  repository may then decide to make this artifact available again as 
  regular OCI artifact.

- **`globalAccess`** (optional) *access method specification*

  If a resource blob is stored locally, the repository implementation
  may decide to provide an external access information (independent
  of the OCM model).

  For example, an OCI artifact stored as local blob
  can be additionally stored as regular OCI artifact in an OCI registry.
  
  This additional external access information can be added using
  a second external access method specification.


### Go Bindings

The go binding can be found [here](method.go)

### Storage Mapping

Transporting component versions by value internalizes externally 
referenced content (for example OCI image references). Those 
resources will then be stored as local blobs using the media type provided by the
original blob.

When importing such a local blob into a repository again, it might be possible
to provide an external access, again. This will be handled
by registered blob handlers.

#### Provided Blob Handlers

The standard tool set uses the following registered blob handlers:
- *Blob handler for importing oci artifact blobs into 
an OCM repository mapped to an OCI registry*

  In this case the oci artifact  blobs will be expanded to a regular 
  OCI artifact taking the optional `referenceName`into account.

Additional blob handlers might be registered by local incarnations.