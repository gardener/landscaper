
# Access Method `ociBlob` - OCI Blob Access


### Synopsis
```
type: ociBlob/v1
```

Provided blobs use the following media type: attribute `mediaType`

### Description
This method implements the access of an OCI blob stored in an OCI repository.

Supported specification version is `v1`



### Specification Versions

#### Version `v1`

The type specific specification fields are:

- **`imageReference`** *string*

  OCI repository reference (this artifact name used to store the blob).

- **`mediaType`** *string*

  The media type of the blob

- **`digest`** *string*

  The digest of the blob used to access the blob in the OCI repository.

- **`size`** *integer*

  The size of the blob


### Go Bindings

The go binding can be found [here](method.go)
