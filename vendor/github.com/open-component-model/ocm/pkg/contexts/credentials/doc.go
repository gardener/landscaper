// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

// Package credentials handles the access to credentials for consumers of
// credential sets.
//
// A credentials set is just a set of simple key/values pair,
// for example username and password.
// Every credential consumer, for example repository implementation of other
// context types, (OCI repositories, OCM repositories, ...) uses the same
// procedure to get to its credentials:
//  1. it composes a most significant typed ConsumerIdentity for every request. This
//     is a set of name/value pairs describing the access context. For an OCI
//     registry, this is for example:
//     - the type (OCIRegistry)
//     - the hostname
//     - an optional port
//     - the repository path
//  2. it then requests credentials from its credentials Context for this consumer.
//  3. the credentials context matches the requested consumer against configured
//     consumers using a dedicated matcher. (For example: finding the consumer
//     specification with the longest matching repository path prefix (for OCI))
//  4. the credentials for the best matching entry are then returned to the requester.
//
// The credentials context is the mediator between credential providers and
// credential consumers. Here
//   - it is possible to explicitly configure credentials for consumer ids
//   - it is possible to manage credential repositories providing named
//     credential sets and
//   - to map dedicated such sets to consumer ids.
//   - specialized credential repositories, may propagate their contained
//     credentials to auto-calculated consumer ids.
//
// This way, there is a special credential repository type DockerConfig. It
// knows what its credentials are meant for (for accessing OCI registries). When
// instantiating such a repository, it automatically exposes its credentials
// under the appropriate consumer ids used by the OCI repository implementation.
// But docker does not allow for separate credentials for different repository
// prefixes in OCI registries (for example organisations in ghcr.io), only per
// host. Therefore, the propagated consumer ids do not provide the path
// property of a consumer id. Together with the path prefix matcher, those id
// settings therefore match all OCI credential requests for all repository paths
// of a dedicated host, as long as there is no more significant setting.
//
// The credentials context also provides a configuration objeect managed by
// a ConfigurationContext and used to configure a credentials context. The
// serialization form of this object can be put into a configuration object of
// the configuration context. For example, the .ocmconfig file is then a
// serialization of such an object which is initially read by the OCM CLI to
// configure the used ConfigurationContext.
// If it describes a credentials configuration this one is applied to the
// credentials context. Such a credentials config object allows to
//   - describe direct consumer id to credential set mappings
//   - describe the instantiation of credential repositories (for example a
//     dockerconfig repo)
//   - the mapping of credential sets of any credential repository to consumer
//     ids (for example mapping of vault entries to consumers (vault not
//     implemented yet)
//
// As for very context type the Context is the central element of this package.
// It provides access to the complete functionality by bundling all the settings
// required to provide credentials to its clients.
package credentials
