// Package inline defines a repository type "inline", that allows to include the whole repository in its specification
// through interpreting yaml as file system.
//
// This package only implements the repository spec and the corresponding serialization and deserialization. The actual
// implementation is provided through conversion of such a repository spec to an internalspec.
package inline
