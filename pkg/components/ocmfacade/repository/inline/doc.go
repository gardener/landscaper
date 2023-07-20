// Package inline provides a READONLY ocm repository implementation for dealing with inline component descriptors and inline
// artifacts (e.g. inline blueprints) through yaml filesystems.
//
// IMPORTANT:
// To configure a filesystems to default to, use compvfs.Set(...), compvfs.Get(...) and blobvfs.Set(...),
// blobvfs.Get(...) respectively.
//
// There are two ways to create a repository, or rather repository spec:
// 1. within the program by calling a NewRepositorySpec function or
// 2. from a serialized specification
//
// The idiomatic way to use the inline repository is to include yaml filesystems in the specification. In this case
// the default filesystem are not necessary or, if set, ignored.
// As currently inline component descriptors and inline blueprints are directly passed to the landscaper through the
// installation, it also has to be possible to pass the filesystems in a different way. One option is to pass them to
// a call to NewRepositorySpec(). The other option is to set attributes in the ocm context through compvfs.Set(...) and
// blobvfs.Set() which will be used if the filesystems are nil during the spec.Repository() call.
package inline
