// Package local defines a repository type "local", that resembles the legacy implementation for interpreting
// directories as ocm repositories. The structure of such a local ocm repository resembles the structure of a
// component archive, although such a "local" repository may contain multiple components.
//
// This package only implements the repository spec and the corresponding serialization and deserialization. The actual
// implementation is provided through conversion of such a repository spec to an internalspec.
package local
