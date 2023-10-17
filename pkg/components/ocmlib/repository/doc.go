// Package repository provides a read-only implementation of an ocm repository.
//
// The repository enables 2 primary usage scenarios:
//   - Using a directory within the file system as an ocm repository. This is especially convenient for local test
//     scenarios.
//   - Using a list of component descriptors available in your program as an ocm repository. This has several potential
//     usage scenarios. In the landscaper, its primary use case is to convert (potentially a tree of) component
//     descriptors provided as inline component descriptors in the installation to an ocm repository, so this special
//     scenario can be treated exactly the same afterwards.
//
// To support a wide range of use cases, the internalspec allows to specify:
//   - A file system within which the component descriptors are stored (FileSystem).
//   - A path within that file system to specify the directory within which the component descriptors are stored
//     (CompDescDirPath).
//   - A file system within which the blobs described by the resources of the component descriptors are stored
//     (BlobFs).
//   - A path within the file system to specify the directory within which the blobs are stored (BlobDirPath).
//   - A mode which currently can be either "filesystem" or "context". This mode is only relevant if no dedicated
//     blob file system (BlobFs) has been specified (and will be ignored otherwise). The default mode is "filesystem"
//     which simply means that the blobs are located in the same file system as the component descriptors (thus,
//     BlobFs = FileSystem). "context" means that the blob file system will be read from the ocm library context. The
//     repository supports a specification type "inline" (see corresponding package) that allows to specify component
//     descriptors inline (through interpreting yaml as file system - not the same as landscaper inline component
//     descriptors!). To be able to describe artifacts that reside in the local file system in that inline component
//     descriptor you can specify that the blob file system shall be taken from the ocmlib context. So the
//     application can decide where local artifacts described in inline component descriptors are read from by setting
//     this context attribute (vfsattr).
package repository
