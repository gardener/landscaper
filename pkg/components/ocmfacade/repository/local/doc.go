// Package local provides a READONLY ocm repository implementation for dealing with component descriptors and artifacts
// in filesystems.
//
// IMPORTANT:
// To configure a filesystem to default to, use localrootfs.Set(...), localrootfs.Get(...).
//
// There are two ways to create a repository, or rather repository spec:
// 1. within the program by calling a NewRepositorySpec function or
// 2. from a serialized specification
//
// In the first case, it is pretty straight forward. One can pass the filesystem instances in which the component
// descriptors and the artifacts are located and from there deal with it as with any other repository.
// In the second case, the spec is created from a serialized specification. As it is usually impractical to serialize
// a local filesystem, the filesystems in the spec are nil. In this case, the filesystems are read from an attribute
// in the ocm context which can be set through localrootfs.Set(...) and read through localrootfs.Get(...) during
// the spec.Repository call (usually, you want this attribute to be equal to the filesystem configured in the
// landscapers local registry config).
package local
