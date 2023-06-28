
# Access Method `relativeOciReference` - OCI Artifact Access in OCI-registry-based OCM Repository


### Synopsis

```
type: relativeOciReference/v1
```

Provided blobs use the following media type:

- `application/vnd.oci.image.manifest.v1+tar+gzip`: OCI image manifests
- `application/vnd.oci.image.index.v1+tar.gzip`: OCI index manifests

Depending on the repository appropriate docker legacy types might be used.

The artifact content is provided in the [Artifact Set Format](../../../oci/repositories/ctf/formatspec.md#artifact-set-archive-format).
The tag is provided as annotation.

### Description

This method implements the access of an OCI artifact stored in an OCI registry,
which is used to host the OCM repository the component version is retrieved from.

It works similar to the [`ociArtifact`](../ociartifact/README.md) access method,
but the used reference does not contain the OCI registry host, which is
taken from the OCI registry used to host the component version containing
the access specification.

Supported specification version is `v1`


### Specification Versions

This access method is a legacy access method formerly used to enable 
physical replication of OCI registry content together with referenced OCI artifacts.

This should basically be done by a value transport of the component versions, because it
is a special case for OCI artifacts stored together with component versions in the same
OCI registry.

#### Version `v1`

The type specific specification fields are:

- **`reference`** *string*

  OCI image/artifact reference following the OCI schemes to describe an arifact inside
  an OCI registry (OCI reference without the host part):
  - `<artifact>:<digest>@<tag>`
  - `<repo path>/<artifact>:<version>@<tag>`

### Go Bindings

The go binding can be found [here](method.go)
