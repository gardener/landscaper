# `s3` - Blobs in a Simple Storage System (S3)


### Synopsis
```
type: s3/v1
```

Provided blobs use the following media type: attribute `mediaType`

### Description

This method implements the access of a blob stored in an S3 bucket.


### Specification Versions

Supported specification version is `v1`

#### Version `v1`

The type specific specification fields are:

- **`region`** (optional) *string*

  OCI repository reference (this artifact name used to store the blob).

- **`bucket`** *string*

  The name of the S3 bucket containing the blob

- **`key`** *string*

  The key of the desired blob


